package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func HandleAddPlaylist(w http.ResponseWriter, r *http.Request) {
	log.Info("received a request to add a playlist")
	w.Header().Set("Content-Type", "application/json")
	var playlistData AddTargetPayload
	var res Response
	err := json.NewDecoder(r.Body).Decode(&playlistData)
	if err != nil {
		ReturnResponse(w, Response{Type: "Error", Key: "ERROR_PARSING_DATA", Message: "There was an error parsing json: " + err.Error()})
	}

	playlist := DownloadTarget{URL: playlistData.URL, Type: "Playlist"}

	doesPlaylistExist, err := playlist.DoesExist()
	if err != nil {
		log.Info("error doesChannelExist: ", err)
		ReturnResponse(w, Response{Type: "Error", Key: "DOES_EXIST_ERROR", Message: "There was an error while trying to check if the channel already exists" + err.Error()})
	}

	if doesPlaylistExist == true {
		log.Info("this playlist already exists")
		ReturnResponse(w, Response{Type: "Success", Key: "PLAYLIST_ALREADY_EXISTS", Message: "This playlists already exists"})
	} else {
		playlistMetadata, err := playlist.GetMetadata()
		if err != nil {
			ReturnResponse(w, Response{Type: "Error", Key: "ERROR_GETTING_METADATA", Message: "There was an error getting channel metadata: " + err.Error()})
		}

		if playlistData.DownloadMode == "Audio Only" {
			playlist = DownloadTarget{URL: playlistData.URL, DownloadMode: playlistData.DownloadMode, Name: playlistMetadata.Playlist, PreferredExtensionForAudio: playlistData.FileExtension, DownloadHistory: []string{}, LastChecked: time.Now().Format("01-02-2006 15:04:05"), CheckingInterval: "", Type: "Playlist"}
		} else if playlistData.DownloadMode == "Video And Audio" {
			playlist = DownloadTarget{URL: playlistData.URL, DownloadMode: playlistData.DownloadMode, Name: playlistMetadata.Playlist, PreferredExtensionForVideo: playlistData.FileExtension, DownloadHistory: []string{}, LastChecked: time.Now().Format("01-02-2006 15:04:05"), CheckingInterval: "", Type: "Playlist"}
		}

		err = playlist.AddToDatabase()
		if err != nil {
			log.Error(err)
			ReturnResponse(w, Response{Type: "Error", Key: "ERROR_ADDING_PLAYLIST", Message: "There was an error adding the playlist to the database" + err.Error()})
		}
		err = playlist.Download(playlistData.DownloadQuality, playlistData.FileExtension, playlistData.DownloadEntire)
		if err != nil {
			res = Response{Type: "Error", Key: "ERROR_DOWNLOADING_ENTIRE_CHANNEL", Message: "There was an error downloading the entire channel" + err.Error()}
			ReturnResponse(w, res)
		}
		err = playlist.UpdateLatestDownloaded(playlistMetadata.ID)
		if err != nil {
			res = Response{Type: "Error", Key: "ERROR_CHECKING_CHANNEL", Message: "There was an error while checking the channel: " + err.Error()}
			ReturnResponse(w, res)
		}
		err = playlist.UpdateDownloadHistory(playlistMetadata.ID)
		if err != nil {
			res = Response{Type: "Error", Key: "ERROR_CHECKING_CHANNEL", Message: "There was an error while checking the channel: " + err.Error()}
			ReturnResponse(w, res)
		}
		res = Response{Type: "Success", Key: "ADD_CHANNEL_SUCCESS", Message: "Channel successfully added and downloaded latest video"}
		ReturnResponse(w, res)
	}
}

func HandleCheckPlaylist(w http.ResponseWriter, r *http.Request) {
	log.Info("received a request to check a playlist for new uploads")
	w.Header().Set("Content-Type", "application/json")
	var data AddTargetPayload
	var res Response
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnResponse(w, Response{Type: "Error", Key: "ERROR_PARSING_DATA", Message: "There was an error parsing json: " + err.Error()})
	}
	playlist := DownloadTarget{URL: data.URL, Type: "Playlist"}
	playlist, _ = playlist.GetFromDatabase()

	newVideoFound, videoId, err := playlist.CheckNow()
	if err != nil {
		res = Response{Type: "Error", Key: "ERROR_CHECKING_PLAYLIST", Message: "There was an error while checking the playlist: " + err.Error()}
	}
	if newVideoFound == true {
		err = playlist.Download("best", data.FileExtension, false)
		if err != nil {
			res = Response{Type: "Error", Key: "ERROR_CHECKING_PLAYLIST", Message: "There was an error while checking the channel: " + err.Error()}
		}
		err = playlist.UpdateLatestDownloaded(videoId)
		if err != nil {
			res = Response{Type: "Error", Key: "ERROR_CHECKING_PLAYLIST", Message: "There was an error while checking the channel: " + err.Error()}
		}
		err = playlist.UpdateDownloadHistory(videoId)
		if err != nil {
			res = Response{Type: "Error", Key: "ERROR_CHECKING_PLAYLIST", Message: "There was an error while checking the channel: " + err.Error()}
		}
		res = Response{Type: "Success", Key: "NEW_VIDEO_DETECTED", Message: "New video detected for " + playlist.Name + " and downloaded"}
	} else {
		res = Response{Type: "Success", Key: "NO_NEW_VIDEOS", Message: "No new videos detected for " + playlist.Name}
	}
	ReturnResponse(w, res)
}

func HandleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	log.Info("received a request to get all playlists")
	w.Header().Set("Content-Type", "application/json")

	playlists, err := GetAll("playlists")
	if err != nil {
		res := Response{Type: "Error", Key: "ERROR_GETTING_PLAYLISTS", Message: "There was an error while getting playlists: " + err.Error()}
		json.NewEncoder(w).Encode(res)
		ReturnResponse(w, Response{Type: "Error", Key: "ERROR_GETTING_CHANNELS", Message: "There was an error while getting playlists: " + err.Error()})
	}
	json.NewEncoder(w).Encode(playlists)
}

func HandleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	log.Info("received a request to delete a playlist")

	w.Header().Set("Content-Type", "application/json")
	var data DeletePlaylistPayload
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnResponse(w, Response{Type: "Error", Key: "ERROR_PARSING_DATA", Message: "There was an error parsing json: " + err.Error()})
	}
	playlistURL := data.URL
	playlistURL = strings.Replace(playlistURL, "delPlaylist", "", -1)
	playlist := DownloadTarget{URL: playlistURL, Type: "Playlist"}

	playlist.Delete()

	ReturnResponse(w, Response{Type: "Success", Key: "DELETE_PLAYLIST_SUCCESS", Message: "Playlist removed"})
}

func HandleCheckAllPlaylists(w http.ResponseWriter, r *http.Request) {
	log.Info("received a request to check all playlists for new uploads")
	w.Header().Set("Content-Type", "application/json")
	res, err := CheckAll("playlists")
	if err != nil {
		ReturnResponse(w, Response{Type: "Error", Key: "ERROR_CHECKING_PLAYLISTS", Message: "There was an error while checking playlists: " + err.Error()})
	}
	ReturnResponse(w, res)
}
