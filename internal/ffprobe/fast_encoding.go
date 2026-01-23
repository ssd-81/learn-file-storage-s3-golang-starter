package ffprobe

import (
	"bytes"
	"fmt"
	"os/exec"
)



func ProcessVideoForFastStart(filePath string) (string, error) {
	
	newFilePath := filePath + ".processing"
	fmt.Println(">>>>>>>", newFilePath, "<<<<<<<<")
	// _, err := os.Create(newFilePath)
	// if err != nil {
	// 	return "", err
	// }

	cmd := exec.Command("ffmpeg", "-i",filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilePath)
	var b bytes.Buffer;
	var e bytes.Buffer;

	// cmd.Stdout
	cmd.Stdout = &b
	cmd.Stderr = &e
	err := cmd.Run()
	if err != nil {
		fmt.Println(e.String())
		return "", err
	}
	return newFilePath, nil 
}