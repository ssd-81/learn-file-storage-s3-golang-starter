package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/ffprobe"

	"github.com/google/uuid"
)

// func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error){

// 	if video.VideoURL == nil {
//         return video, nil
//     }
//     parts := strings.Split(*video.VideoURL, ",")
//     if len(parts) < 2 {
//         return video, nil
//     }
// 	fmt.Println("raw video URL:", *video.VideoURL)
// 	fmt.Println("parts:", parts)
// 	bucket, key := parts[0], parts[1] // double check if these values are as expected
// 	url, err := upload.GeneratePresignedURL(cfg.s3Client, bucket, key, 5 * time.Minute)
// 	if err != nil {
// 		fmt.Println("error encountered (GeneratePresignedURL)", err)
// 		return video, err // not certain if the same video object should be returned
// 	}
// 	video.VideoURL = &url
// 	return video, nil

// }


func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// 1 << 30 bytes 
	// http.MaxBytesReader
	const maxMemory = 1 << 30

	// not sure how to use this at the moment
	// r.Body is being used a limited reader 
	r.Body = http.MaxBytesReader(w, r.Body , maxMemory) 

	vidId, err := uuid.Parse(r.PathValue("videoID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(vidId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "video not found", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "video is not owned by user", fmt.Errorf("user does not own the video"))
		return 
	}


	// try to understand this block 
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 400, "invalid thumbnail data", err)
		return
	}
	defer file.Close()
	// 


	// did not initialize params due to error 
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid mediaType", err)
		return 
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusNotAcceptable, "invalid media type", fmt.Errorf("only video/mp4 allowed"))
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-video.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "video could not be created (temp)", err)
		return 
	}
	
	

	defer os.Remove(tempFile.Name()) // check on this method; why "tubely-video.mp4" can't be used
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not move files from wire to temp files", err)
		return 
	}
	aspRatio , err := ffprobe.GetVideoAspectRatio(tempFile.Name())
	fmt.Println(">>>>>>", aspRatio, "<<<<<<<")
	var view string 

	switch aspRatio {
	case "16:9":
		view = "landscape"
	case "9:16":
		view = "portrait"
	default:
		view = "other"
	}

	fmt.Println("file copied to local system")
	tempFile.Seek(0, io.SeekStart) // setting the file's pointer back to beginning
	newPath, err := ffprobe.ProcessVideoForFastStart(tempFile.Name())
	if err != nil {
		log.Printf("error while converting video to fast video")
		respondWithError(w, http.StatusInternalServerError, "fast encoding conversion failed", err)
		return 
	}
	defer os.Remove(newPath)


	fastFile, err := os.Open(newPath)
	if err != nil {
		log.Printf("fast encoded file could not be opened")
		respondWithError(w, http.StatusInternalServerError, "fast encoded file could not be opened", err)
		return 
	}
	
	// checking if newly created file is empty
	fastFileInfo, err:= fastFile.Stat()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "newly created fast file info not found", err)
		return 
	}
	fmt.Println("the size of the file is: ")
	fmt.Println(fastFileInfo.Size())


	// generating a random 32 bit integer filename 
	key := make([]byte, 32)
	rand.Read(key)
	buf := &bytes.Buffer{}
	encoder := base64.NewEncoder(base64.RawURLEncoding, buf)
	encoder.Write(key)
	encoder.Close()


	fName := fmt.Sprintf("%v/%v.%v", view, buf.String(), "mp4") // hardcoded mp4 for now; check handler_upload_thumbnail to make changes
	inputParams := s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &fName,
		Body: fastFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &inputParams)
	if(err != nil) {
		fmt.Println("PutObject ERROR:", err)
		respondWithError(w, http.StatusInternalServerError, "Error uploading file to S3", err)
		return
	}
	
	// updatedUrl := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, fName)
	// made a blunder by passing memory address rather than actual value 
	// and I don't know why go accepted that in the first place; my mistake completely
	updatedUrl := fmt.Sprintf("%v,%v", cfg.s3Bucket, *inputParams.Key)
	video.VideoURL = &updatedUrl // unused write; check that on again 
	// video, err = cfg.dbVideoToSignedVideo(video)
	
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "video url not updated", err)
		return 
	}
}


