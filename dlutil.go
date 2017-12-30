package main

import (
	"context"
	"fmt"

	"github.com/dfordsoft/golib/ebook"
	"golang.org/x/sync/semaphore"
)

type contentUtil struct {
	index   int
	title   string
	link    string
	content string
}
type downloadUtil struct {
	downloader func(string) []byte
	generator  ebook.IBook

	currentPage int
	maxPage     int
	quit        chan bool
	content     chan contentUtil
	buffer      []contentUtil
	ctx         context.Context
	semaphore   *semaphore.Weighted
}

func (du *downloadUtil) init() {
	du.quit = make(chan bool)
	du.ctx = context.TODO()
	du.semaphore = semaphore.NewWeighted(10)
	du.content = make(chan contentUtil, 10000)
}

func (du *downloadUtil) wait() {
	<-du.quit
}

func (du *downloadUtil) addURL(index int, title string, link string) {
	// semaphore
	du.semaphore.Acquire(du.ctx, 1)
	go func() {
		du.content <- contentUtil{
			index:   index,
			title:   title,
			link:    link,
			content: string(du.downloader(link)),
		}
		du.semaphore.Release(1)
	}()
}

func (du *downloadUtil) process() {
	go func() {
		for {
			select {
			case cu := <-du.content:
				fmt.Println(cu.title, cu.link)
				if cu.index == du.currentPage+1 {
					du.generator.AppendContent(cu.title, cu.link, cu.content)
					du.currentPage++
				} else {
					// insert into local buffer
					if len(du.buffer) == 0 || du.buffer[0].index > cu.index {
						// push front
						du.buffer = append([]contentUtil{cu}, du.buffer...)
						break
					}
					if du.buffer[len(du.buffer)-1].index < cu.index {
						// push back
						du.buffer = append(du.buffer, cu)
						break
					}
					for i := 0; i < len(du.buffer)-1; i++ {
						if du.buffer[i].index < cu.index && du.buffer[i+1].index > cu.index {
							// insert at i+1
							du.buffer = append(du.buffer[:i+1], append([]contentUtil{cu}, du.buffer[i+1:]...)...)
							break
						}
					}
					break
				}

				// check local buffer to pick items to generator
				for ; len(du.buffer) > 0 && du.buffer[0].index == du.currentPage+1; du.buffer = du.buffer[1:] {
					du.generator.AppendContent(du.buffer[0].title, du.buffer[0].link, du.buffer[0].content)
					du.currentPage++
				}
				if du.currentPage == du.maxPage {
					du.quit <- true
					return
				}
			}
		}
	}()
}
