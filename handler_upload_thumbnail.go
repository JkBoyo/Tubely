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
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	const maxMem = 10 << 20
	err := r.ParseMultipartForm(maxMem)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}
	mPF, mPFH, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File and header weren't found", err)
		return
	}
	medTyp := mPFH.Header.Get("Content-Type")

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
	vidDat, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't fetch video data", err)
		return
	}
	if vidDat.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not your video", errors.New("User doesn't own video"))
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	mimeT, _, err := mime.ParseMediaType(medTyp)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File type is wrong", err)
		return
	}

	var fileExt string
	switch mimeT {
	case "image/jpeg":
		fileExt = "jpg"
	case "image/png":
		fileExt = "png"
	default:
		respondWithError(w, http.StatusBadRequest, "Wrong file type", err)
		return
	}

	fileNameBytes := make([]byte, 32)
	rand.Read(fileNameBytes)

	fileName := base64.RawURLEncoding.EncodeToString(fileNameBytes)

	thumbFP := filepath.Join("./assets", fmt.Sprintf("%v.%v", fileName, fileExt))

	file, err := os.Create(thumbFP)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "File creation error", err)
		return
	}

	defer file.Close()

	io.Copy(file, mPF)

	thumbUrl := fmt.Sprintf("http://localhost:%v/%v", cfg.port, thumbFP)

	vidDat.ThumbnailURL = &thumbUrl

	cfg.db.UpdateVideo(vidDat)

	respondWithJSON(w, http.StatusOK, vidDat)
}
