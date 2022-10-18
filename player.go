package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/vansante/go-ffprobe.v2"

	stream_db "github.com/tech7strann1k/online-radio/db"
)

var url string

type StreamPlayer struct {
	StreamName, StreamLogo, StreamUrl, 
	playing_state, record_state 			string
	StreamList 								[]stream_db.StreamItem
	Started, Playing, ItemChanged, 
	MetadataUpdated, 
	StopUpdatingMetadata,  Stopped 			string
	playCommand, recordCommand				*exec.Cmd
}

func NewPlayer() *StreamPlayer {
	player := &StreamPlayer{Started: "started",
							Playing: "playing", 
							ItemChanged: "item_changed", 
							MetadataUpdated: "metadata_updated",
							StopUpdatingMetadata: "stop_updating_metadata",
							Stopped: "stopped"}
	return player
}

func (player *StreamPlayer) Play() *StreamPlayer {
	comm := exec.Command("ffplay", "-nodisp", "-i", player.StreamUrl)
	err := comm.Start()
	if err != nil {
		fmt.Println(err)
	}
	player.playCommand = comm
	return player
}

func (player *StreamPlayer) StopPlayback() error {
	time.Sleep(time.Duration(time.Duration.Milliseconds(3)))
	_ = player.playCommand.Process.Kill()
	return player.playCommand.Wait()
}

func (player *StreamPlayer) StopRecording() error {
	time.Sleep(time.Duration(time.Duration.Milliseconds(3)))
	_ = player.recordCommand.Process.Kill()
	return player.recordCommand.Wait()
}

func (player *StreamPlayer) GetMetadata(out chan <- *ffprobe.Format) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancelFn()
	ffprobe.SetFFProbeBinPath("/usr/bin/ffprobe")
	data, err := ffprobe.ProbeURL(ctx, player.StreamUrl)
	if err != nil {
		fmt.Println(err)
	}
	select {
		case <-ctx.Done():
			fmt.Println(ctx.Err())
		case out <- data.Format:
	}
}

func (player *StreamPlayer) RecordStream()  {	
	var path = fmt.Sprintf("%s/%s", homeDir, "Музыка")
	var filename = fmt.Sprintf("%s/%s", path, "rec-01.mp3")
	filelist, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	pattern := `rec-(\d+).mp3`
	exp, _ := regexp.Compile(pattern)
	for _, file := range filelist {
		if exp.MatchString(file.Name()) {
			m := exp.FindStringSubmatch(file.Name())
			chars := strings.Split(m[1], "")
			if  chars[0] == "0" {
				if chars[1] == "9" {
					filename = fmt.Sprintf("%s/%s", path, "rec-10.mp3")
				} else {
					num, _ := strconv.Atoi(chars[1])
					num++
					filename = fmt.Sprintf("%s/%s%d%s", path, "rec-0", num, ".mp3")	
				}
			} else {
				num, _ := strconv.Atoi(m[1])
				num++
				filename = fmt.Sprintf("%s/%s%d%s", path, "rec-", num, ".mp3")	
			}
		}
	}
	if player.record_state == "" || player.record_state == "recorded" {
		player.record_state = "recording"
		player.recordCommand = exec.Command("ffmpeg", "-y", "-i", player.StreamUrl, filename)
		player.recordCommand.Start()
	} else {
		player.record_state = "recorded"
		err = player.StopRecording()
		if err != nil {
			fmt.Println("Error recording", err)
		}
	}
}