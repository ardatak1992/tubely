package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

func getVideoAspectRatio(filepath string) (string, error) {

	const aspectRatioTolerance = 0.02
	const aspect16_9 = 1.77
	const aspect9_16 = 0.5625

	type res struct {
		Streams []struct {
			Width  int32 `json:"width"`
			Height int32 `json:"height"`
		} `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)

	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var size res

	err = json.Unmarshal(stdoutBuf.Bytes(), &size)
	if err != nil {
		return "", err
	}

	aspectRatio := float32(size.Streams[0].Width) / float32(size.Streams[0].Height)

	if aspectRatio > aspect16_9-aspectRatioTolerance &&
		aspectRatio < aspect16_9+aspectRatioTolerance {
		return "16:9", nil
	}

	if aspectRatio > aspect9_16-aspectRatioTolerance &&
		aspectRatio < aspect9_16+aspectRatioTolerance {
		return "9:16", nil
	}

	return "other", nil

}
