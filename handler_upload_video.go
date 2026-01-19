package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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


	fmt.Println("file copied to local system")
	tempFile.Seek(0, io.SeekStart) // setting the file's pointer back to beginning

	// generating a random 32 bit integer filename 
	key := make([]byte, 32)
	rand.Read(key)
	buf := &bytes.Buffer{}
	encoder := base64.NewEncoder(base64.RawURLEncoding, buf)
	encoder.Write(key)
	encoder.Close()


	fName := fmt.Sprintf("%v.%v", buf.String(), "mp4") // hardcoded mp4 for now; check handler_upload_thumbnail to make changes
	inputParams := s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &fName,
		Body: tempFile,
		ContentType: &mediaType,
	}
	cfg.s3Client.PutObject(r.Context(), &inputParams)
	// make sure it is fName and not fName
	updatedUrl := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, fName)
	video.VideoURL = &updatedUrl // unused write; check that on again 
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "video url not updated", err)
		return 
	}
}