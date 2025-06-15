package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filepath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	buf := bytes.Buffer{}
	cmd.Stdout = &buf

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	type VidInfo struct {
		Streams []struct {
			Width  int `json:"width,omitempty"`
			Height int `json:"height,omitempty"`
		} `json:"streams"`
	}

	vidInfo := VidInfo{}
	err = json.Unmarshal(buf.Bytes(), &vidInfo)
	if err != nil {
		return "", err
	}

	//r, redWidth, redHeight := gcd(vidInfo.Streams[0].Width, vidInfo.Streams[0].Height)

	//fmt.Printf("Remainder: %d, Reduced Width: %d, Reduced Height %d", r, redWidth, redHeight)
	//aspectRatio := fmt.Sprintf("%v:%v", redWidth, redHeight)
	h, w := vidInfo.Streams[0].Height, vidInfo.Streams[0].Width
	if int(w/h) == int(16/9) {
		return "landscape", nil
	} else if int(w/h) == int(9/16) {
		return "portrait", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filepath string) (string, error) {
	outputFilepath := filepath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filepath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilepath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputFilepath, nil
}

type eEA struct {
	Q int
	R int
	S int
	T int
}

func gcd(a, b int) (r, redW, redH int) {
	euclid := []eEA{
		{R: a, S: 1, T: 0},
		{R: b, S: 0, T: 1},
	}

	for i := 2; ; i++ {

		next := eEA{}
		next.Q = euclid[i-2].R / euclid[i-1].R
		next.R = euclid[i-2].R - next.Q*euclid[i-1].R
		next.S = euclid[i-2].S - next.Q*euclid[i-1].S
		next.T = euclid[i-2].T - next.Q*euclid[i-1].T
		if next.R == 0 {
			return euclid[i-1].R, int(math.Abs(float64(next.T))), int(math.Abs(float64(next.S)))
		}
		euclid = append(euclid, next)
	}
}
