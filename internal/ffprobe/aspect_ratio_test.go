package ffprobe


import "testing"


// /home/ayanami/workspace/github.com/ssd-81/learn-file-storage-s3-golang-starter/samplesboots-video-vertical.mp4
func TestGetVideoAspectRatio(t *testing.T) {
	result, _ := GetVideoAspectRatio("/home/ayanami/workspace/github.com/ssd-81/learn-file-storage-s3-golang-starter/samples/boots-video-vertical.mp4")

	t.Log(result)
}