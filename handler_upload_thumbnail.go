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
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {

	fmt.Println("assetsRoot:", cfg.assetsRoot)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// 10 << 20 = 10 * 1024 * 1024
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 400, "invalid thumbnail data", err)
		return
	}
	defer file.Close()

	// ct := r.Header.Get("Content-Type")
	// ct := header.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid mediaType", err)
		return 
	}
	fmt.Println("mediatype:", mediaType)

	// temp := strings.Split(ct, "/")
	// if len(temp) != 2 {
	// 	respondWithError(w, http.StatusNotAcceptable, "invalid content-type", fmt.Errorf("invalid content type"))
	// 	return
	// }
	// if temp[0] != "image" {
	// 	respondWithError(w, http.StatusNotAcceptable, "not image", fmt.Errorf("not image type"))
	// 	return
	// }
	// extension := temp[1] // this is the file type extension we will use
	// fmt.Println("extension: ", extension)

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusNotAcceptable, "invalid media type", fmt.Errorf("only image/jpeg and image/png allowed"))
		return
	}
	extension := strings.Split(mediaType, "/")[1]
	// instead of the videoID, we will generate a random string by first initializing a 32-byte slice and filling it will
	// randomized values
	// var tempSlice []byte
	key := make([]byte, 32)
	rand.Read(key)
	// base64.RawURLEncoding : this is the URL encoding standard we are using
	buf := &bytes.Buffer{} // what is really going on here? why are we using a pointer? WHYYYYY
	encoder := base64.NewEncoder(base64.RawURLEncoding, buf)
	encoder.Write(key)
	encoder.Close()


	subF := fmt.Sprintf("%v.%v", buf.String(), extension)
	fp := filepath.Join(cfg.assetsRoot, subF)

	fpObj, err := os.Create(fp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "file could not be created", err)
		return
	}
	defer fpObj.Close()

	if _, err := io.Copy(fpObj, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error while copying file", err)
		return
	}

	// create a data url: data:<media-type>;base64,<data>
	// made changes to the below line; buf.String() was videoID
	tnUrl := fmt.Sprintf("http://localhost:%s/assets/%v.%v", cfg.port, buf.String(), extension)
	// tnUrl := fmt.Sprintf("data:image/jpeg;base64,%v", imgB64) // not required anymore

	vmd, err := cfg.db.GetVideo(videoID)

	if err != nil {
		respondWithError(w, 400, "failed to get video with the provided id", err)
		return
	}
	if vmd.UserID != userID {
		log.Print(vmd.UserID)
		log.Print(userID)
		respondWithError(w, http.StatusUnauthorized, "unauthorized endpoint", fmt.Errorf("unauthorized"))
		return
	}

	vidParams := database.Video{}
	vidParams = vmd
	vidParams.ID = videoID
	// craft the thumnail url
	vidParams.ThumbnailURL = &tnUrl
	err = cfg.db.UpdateVideo(vidParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "video could not be updated", err)
		return
	}

	respondWithJSON(w, http.StatusOK, params)
}
