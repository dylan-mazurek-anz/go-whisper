package main

import (
	"fmt"
	"log"
	"time"
	// Packages
)

type DownloadCmd struct {
	Model string `arg:"" help:"Model to download"`
}

func (cmd *DownloadCmd) Run(ctx *Globals) error {
	t := time.Now()
	model, err := ctx.service.DownloadModel(ctx.ctx, cmd.Model, func(curBytes, totalBytes uint64) {
		if time.Since(t) > time.Second {
			pct := float64(curBytes) / float64(totalBytes) * 100
			log.Printf("Downloaded %.0f%%", pct)
			t = time.Now()
		}
	})
	if err != nil {
		return err
	}
	fmt.Println(model)
	return nil
}
