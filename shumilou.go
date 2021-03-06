package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/missdeer/getnovel/ebook"
	"github.com/missdeer/golib/httputil"
	"github.com/missdeer/golib/ic"
)

var (
	httpHeadersShumilou = http.Header{
		"Referer":                   []string{"http://www.shumil.com/"},
		"User-Agent":                []string{"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0"},
		"Accept":                    []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"},
		"Accept-Language":           []string{`en-US,en;q=0.8`},
		"Upgrade-Insecure-Requests": []string{"1"},
	}
)

func downloadShumilouPage(u string) (c []byte) {
	var err error
	c, err = httputil.GetBytes(u, httpHeadersShumilou, time.Duration(opts.Timeout)*time.Second, opts.RetryCount)
	if err != nil {
		return
	}
	c = ic.Convert("gbk", "utf-8", c)
	c = bytes.Replace(c, []byte("\r\n"), []byte(""), -1)
	c = bytes.Replace(c, []byte("\r"), []byte(""), -1)
	c = bytes.Replace(c, []byte("\n"), []byte(""), -1)
	leadingStr := "<div style=\"width:336px;overflow:hidden;float:left;\"><script>readc()</script></div>		</div>"
	idx := bytes.Index(c, []byte(leadingStr))
	if idx > 1 {
		c = c[idx+len(leadingStr):]
	}
	endingStr := "<p><b>"
	idx = bytes.Index(c, []byte(endingStr))
	if idx > 1 {
		c = c[:idx]
	}
	c = bytes.Replace(c, []byte("<br /><br />&nbsp;&nbsp;&nbsp;&nbsp;"), []byte("</p><p>"), -1)
	c = bytes.Replace(c, []byte("&nbsp;&nbsp;&nbsp;&nbsp;"), []byte(""), -1)
	c = bytes.Replace(c, []byte("<p>　　"), []byte("<p>"), -1)
	return
}

func downloadShumilou(u string, gen ebook.IBook) {
	b, err := httputil.GetBytes(u, httpHeadersShumilou, time.Duration(opts.Timeout)*time.Second, opts.RetryCount)
	if err != nil {
		return
	}

	b = bytes.Replace(b, []byte("</li><li class=\"zl\">"), []byte("</li>\n<li class=\"zl\">"), -1)

	gen.Begin()

	dlutil := newDownloadUtil(downloadShumilouPage, gen)
	dlutil.process()

	var title string
	index := 0
	// 	<li class="zl"><a href="12954102.html">阅读指南（重要，必读）</a></li>
	r, _ := regexp.Compile(`<li class="zl"><a\shref="([0-9]+\.html)">([^<]+)</a></li>`)
	// <div class="tit"><b>1号球王最新章节列表</b></div>
	re, _ := regexp.Compile(`<div class="tit"><b>([^<]+)</b></div>`)
	scanner := bufio.NewScanner(bytes.NewReader(b))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		// convert from gbk to UTF-8
		l := ic.ConvertString("gbk", "utf-8", line)
		if title == "" {
			ss := re.FindAllStringSubmatch(l, -1)
			if len(ss) > 0 && len(ss[0]) > 0 {
				s := ss[0]
				title = s[1]
				idx := strings.Index(title, `最新章节`)
				if idx > 0 {
					title = title[:idx]
				}
				gen.SetTitle(title)
				continue
			}
		}
		if r.MatchString(l) {
			ss := r.FindAllStringSubmatch(l, -1)
			s := ss[0]
			finalURL := fmt.Sprintf("%s%s", u, s[1])
			index++
			if dlutil.addURL(index, s[2], finalURL) {
				break
			}
		}
	}
	dlutil.wait()
	gen.End()
}

func init() {
	registerNovelSiteHandler(&novelSiteHandler{
		Title:         `书迷楼`,
		MatchPatterns: []string{`http://www.shumil.com/[a-zA-Z0-9]+/`},
		Download:      downloadShumilou,
	})
}
