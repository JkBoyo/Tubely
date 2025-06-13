package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
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
	http.MaxBytesReader(w, r.Body, 1<<30)

	videoIdString := r.PathValue("videoID")

	videoId, err := uuid.Parse(videoIdString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't parse uuid", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	vidDat, err := cfg.db.GetVideo(videoId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't fetch video data", err)
		return
	}

	if vidDat.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not your video", errors.New("User doesn't own video"))
		return
	}

	mPF, mPFH, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File and header weren't found", err)
		return
	}

	defer mPF.Close()

	fmt.Println("uploading video", videoId, "by user", userID)

	medTypStr := mPFH.Header.Get("Content-Type")

	medTyp, _, err := mime.ParseMediaType(medTypStr)

	if medTyp != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "File type is wrong", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	io.Copy(tempFile, mPF)

	tempFile.Seek(0, io.SeekStart)

	fileNameBytes := make([]byte, 32)
	rand.Read(fileNameBytes)

	fileName := base64.RawURLEncoding.EncodeToString(fileNameBytes)

	fileNameExt := fileName + ".mp4"

	tempFileInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileNameExt,
		Body:        tempFile,
		ContentType: &medTyp,
	}

	vidUrl := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, fileNameExt)

	vidDat.VideoURL = &vidUrl

	cfg.db.UpdateVideo(vidDat)

	cfg.s3Client.PutObject(r.Context(), &tempFileInput)
}
