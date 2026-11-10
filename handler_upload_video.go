package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	type response struct {
		key      string
		videoURL string
	}

	http.MaxBytesReader(w, r.Body, 1<<30)
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error uploading video", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	_, err = auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	file, _, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error uploading video", err)
		return
	}
	defer file.Close()

	fileType, _, err := mime.ParseMediaType("video/mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Tried to upload wrong video format", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error uploading video", err)
		return
	}

	key := make([]byte, 32)
	rand.Read(key)

	fileName := base64.URLEncoding.EncodeToString(key) + ".mp4"

	io.Copy(tempFile, file)
	tempFile.Seek(0, io.SeekStart)

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error getting aspect ratio", err)
		return
	}

	var prefix string

	switch aspectRatio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	default:
		prefix = "other"
	}

	fileName = prefix + "/" + fileName

	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileName)

	oldVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error uploading video", err)
		return
	}

	cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileName,
		Body:        tempFile,
		ContentType: &fileType,
	})

	err = cfg.db.UpdateVideo(database.Video{
		ID:                videoID,
		UpdatedAt:         time.Now().UTC(),
		CreatedAt:         oldVideo.CreatedAt,
		CreateVideoParams: oldVideo.CreateVideoParams,
		VideoURL:          &videoURL,
	})

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error uploading video", err)
		return
	}

	respondWithJSON(w, 200, response{
		key:      fileName,
		videoURL: videoURL,
	})
}
