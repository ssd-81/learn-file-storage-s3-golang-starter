package ffprobe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type FFProbeOutput struct {
    Streams []struct {
        Width  int `json:"width"`
        Height int `json:"height"`
    } `json:"streams"`
}

func GetVideoAspectRatio(filePath string) (string, error) {
	// ffprobe -v error -print_format json -show_streams boots-video-horizontal.mp4
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var b bytes.Buffer;
	var e bytes.Buffer;

	// cmd.Stdout
	cmd.Stdout = &b
	cmd.Stderr = &e

	err := cmd.Run()
	
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(e.String())
		return "", err
	}

	var target FFProbeOutput

    err = json.Unmarshal([]byte(b.Bytes()), &target)
    if err != nil {
        // log.Fatalf("Unable to marshal JSON due to %s", err)
		fmt.Println(err)
    }
	fmt.Println(">>>>>>>>")
	fmt.Println(target)

	if len(target.Streams) == 0 {
		 return "", fmt.Errorf("ffprobe output does not contain the aspect ratio")
	}
	first := target.Streams[0]
	h := first.Height
	w := first.Width
	temp := float32(w) / float32(h)
	fmt.Println(">>>>>>", temp, "<<<<<<<<")
	var aspRatio string
	if(temp >= 1.7 && temp < 1.8) {
		aspRatio = "16:9"
	}else if (temp > float32(0.55) && temp < 0.57) {
		aspRatio = "9:16"
	}else {
		aspRatio = "other"
	}
	fmt.Println(aspRatio)

	return aspRatio, nil 
}