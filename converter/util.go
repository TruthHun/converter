package converter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

//media-type
var MediaType = map[string]string{
	".jpeg":  "image/jpeg",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".gif":   "image/gif",
	".ico":   "image/x-icon",
	".bmp":   "image/bmp",
	".html":  "application/xhtml+xml",
	".xhtml": "application/xhtml+xml",
	".htm":   "application/xhtml+xml",
	".otf":   "application/x-font-opentype",
	".ttf":   "application/x-font-ttf",
	".js":    "application/x-javascript",
	".ncx":   "x-dtbncx+xml",
	".txt":   "text/plain",
	".xml":   "text/xml",
	".css":   "text/css",
}

//根据文件扩展名，获取media-type
func GetMediaType(ext string) string {
	if mt, ok := MediaType[strings.ToLower(ext)]; ok {
		return mt
	}
	return ""
}

//解析配置文件
func parseConfig(configFile string) (cfg Config, err error) {
	var b []byte
	if b, err = ioutil.ReadFile(configFile); err == nil {
		err = json.Unmarshal(b, &cfg)
	}
	return
}

//<navPoint id="uypBSBpbQHAkc2dM2WoMbaA" playOrder="11">
//<navLabel>
//<text>2.3 Controller运行机制</text>
//</navLabel>
//<content src="html/11.html"/>
//</navPoint>
func getNavPoint(toc Toc, idx int) (navpoint string, nextidx int) {
	navpoint = `
	<navPoint id="id%v" playOrder="%v">
		<navLabel>
			<text>%v</text>
		</navLabel>
		<content src="%v"/>`
	navpoint = fmt.Sprintf(navpoint, toc.Id, idx, toc.Title, toc.Link)
	nextidx = idx + 1
	return
}

//将toc转成toc.ncx文件
func tocToXml(tocs []Toc, pid, idx int) (codes []string, next_idx int) {
	var code string
	for _, toc := range tocs {
		if toc.Pid == pid {
			code, idx = getNavPoint(toc, idx)
			codes = append(codes, code)
			for _, item := range tocs {
				if item.Pid == toc.Id {
					code, idx = getNavPoint(item, idx)
					codes = append(codes, code)
					var code_arr []string
					code_arr, idx = tocToXml(tocs, item.Id, idx)
					codes = append(codes, code_arr...)
					codes = append(codes, `</navPoint>`)
				}
			}
			codes = append(codes, `</navPoint>`)
		}
	}
	next_idx = idx
	return
}
