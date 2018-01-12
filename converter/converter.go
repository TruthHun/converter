package converter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/gotil/filetil"
)

type Converter struct {
	BasePath       string
	Config         Config
	GeneratedCover string
}

//目录结构
type Toc struct {
	Id    int    `json:"id"`
	Link  string `json:"link"`
	Pid   int    `json:"pid"`
	Title string `json:"title"`
}

//config.json文件解析结构
type Config struct {
	Contributor string   `json:"contributor"`
	Cover       string   `json:"cover"`
	Creator     string   `json:"creator"`
	Timestamp   string   `json:"date"`
	Description string   `json:"description"`
	Footer      string   `json:"footer"`
	Header      string   `json:"header"`
	Identifier  string   `json:"identifier"`
	Language    string   `json:"language"`
	Publisher   string   `json:"publisher"`
	Title       string   `json:"title"`
	Format      []string `json:"format"`
	Toc         []Toc    `json:"toc"`
}

var (
	output = "output" //文档导出文件夹
)

//根据json配置文件，创建文档转化对象
func NewConverter(configFile string) (converter *Converter, err error) {
	var (
		cfg      Config
		basepath string
	)
	if cfg, err = parseConfig(configFile); err == nil {
		if basepath, err = filepath.Abs(filepath.Dir(configFile)); err == nil {
			converter = &Converter{
				Config:   cfg,
				BasePath: basepath,
			}
		}

	}
	return
}

func (this *Converter) Convert() (err error) {
	//最后移除创建的多余而文件
	defer this.converterDefer()

	//创建导出文件夹
	if err = os.Mkdir(output, os.ModePerm); err != nil {
		return err
	}

	return
}

//删除生成导出文档而创建的文件
func (this *Converter) converterDefer() {
	//删除不必要的文件
	go os.RemoveAll(this.BasePath + "/META-INF")
	go os.RemoveAll(this.BasePath + "/mimetype")
	go os.RemoveAll(this.BasePath + "/toc.ncx")
	go os.RemoveAll(this.BasePath + "/content.opf")
	//封面图片待优化
	go os.RemoveAll(this.BasePath + "/titlepage.xhtml")
}

//生成metainfo
func (this *Converter) generateMetaInfo() (err error) {
	xml := `<?xml version="1.0"?>
			<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
			   <rootfiles>
				  <rootfile full-path="content.opf" media-type="application/oebps-package+xml"/>
			   </rootfiles>
			</container>
    `
	folder := this.BasePath + "/META-INF"
	if err = os.MkdirAll(folder, os.ModePerm); err == nil {
		err = ioutil.WriteFile(folder+"/container.xml", []byte(xml), os.ModePerm)
	}
	return
}

//形成mimetyppe
func (this *Converter) generateMimeType() (err error) {
	return ioutil.WriteFile(this.BasePath+"/mimetype", []byte("application/epub+zip"), os.ModePerm)
}

//生成封面
func (this *Converter) generateTitlePage() (err error) {
	if ext := strings.ToLower(filepath.Ext(this.Config.Cover)); !(ext == ".html" || ext == ".xhtml") {
		xml := `<?xml version='1.0' encoding='utf-8'?>
				<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="` + this.Config.Language + `">
					<head>
						<meta http-equiv="Content-Type" content="text/html; charset=UTF-8"/>
						<meta name="calibre:cover" content="true"/>
						<title>Cover</title>
						<style type="text/css" title="override_css">
							@page {padding: 0pt; margin:0pt}
							body { text-align: center; padding:0pt; margin: 0pt; }
						</style>
					</head>
					<body>
						<div>
							<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="100%" height="100%" viewBox="0 0 800 1068" preserveAspectRatio="none">
								<image width="800" height="1068" xlink:href="` + strings.Trim(this.Config.Cover, "./") + `"/>
							</svg>
						</div>
					</body>
				</html>
		`
		if err = ioutil.WriteFile(this.BasePath+"/titlepage.xhtml", []byte(xml), os.ModePerm); err == nil {
			this.GeneratedCover = "titlepage.xhtml"
		}
	}
	return
}

//生成文档目录
func tocNcx(title string, toc []Toc, basepath string) {
	ncx := `<?xml version='1.0' encoding='utf-8'?>
			<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1" xml:lang="zh-CN">
			  <head>
				<meta content="4" name="dtb:depth"/>
				<meta content="calibre (2.85.1)" name="dtb:generator"/>
				<meta content="0" name="dtb:totalPageCount"/>
				<meta content="0" name="dtb:maxPageNumber"/>
			  </head>
			  <docTitle>
				<text>%v</text>
			  </docTitle>
			  <navMap>%v</navMap>
			</ncx>
			`
	codes, _ := tocToXml(toc, 0, 1)
	ncx = fmt.Sprintf(ncx, title, strings.Join(codes, ""))
	if err := ioutil.WriteFile(basepath+"/toc.ncx", []byte(ncx), os.ModePerm); err != nil {
		panic(err)
	}
}

//倒数第二步生成opf
func ContentOpf(book Config, basePath string) {
	guide := ``
	manifest := ``
	meta := `
		<dc:title>%v</dc:title>
		<dc:contributor opf:role="bkp">%v</dc:contributor>
		<dc:publisher>%v</dc:publisher>
		<dc:description>%v</dc:description>
		<dc:language>%v</dc:language>
		<dc:creator opf:file-as="Unknown" opf:role="aut">%v</dc:creator>
		<meta name="calibre:timestamp" content="%v"/>
	`
	meta = fmt.Sprintf(meta, book.Title, book.Contributor, book.Publisher, book.Description, book.Language, book.Creator, book.Timestamp)
	if len(book.Cover) > 0 {
		meta = meta + `<meta name="cover" content="cover"/>`
		guide = `<reference href="titlepage.xhtml" title="Cover" type="cover"/>`
		manifest = fmt.Sprintf(`<item href="%v" id="cover" media-type="%v"/>`, strings.Trim(book.Cover, "./"), GetMediaType(filepath.Ext(book.Cover)))
	}

	spine := ``
	//扫描所有文件
	if files, err := filetil.ScanFiles(basePath); err == nil {

		manifestArr := []string{}
		spineArr := []string{}
		for _, file := range files {
			if !file.IsDir && file.Name != "book.json" {
				id := cryptil.Md5Crypt(file.Path)
				ext := strings.ToLower(filepath.Ext(file.Path))
				basePath = strings.Replace(basePath, "\\", "/", -1)
				sourcefile := strings.TrimPrefix(file.Path, basePath+"/")
				if sourcefile != strings.TrimLeft(book.Cover, "./") {
					manifestArr = append(manifestArr,
						fmt.Sprintf(`<item href="%v" id="%v" media-type="%v"/>`, sourcefile, id, GetMediaType(ext)),
					)
					if ext == ".html" || ext == ".xhtml" {
						spineArr = append(spineArr, fmt.Sprintf(`<itemref idref="%v"/>`, id))
					}
				}

			}
		}
		manifest = manifest + strings.Join(manifestArr, "")
		spine = strings.Join(spineArr, "")
	} else {
		panic(err)
	}

	pkg := `<?xml version='1.0' encoding='utf-8'?>
		<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="uuid_id" version="2.0">
		  <metadata xmlns:opf="http://www.idpf.org/2007/opf" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:calibre="http://calibre.kovidgoyal.net/2009/metadata">
			%v
		  </metadata>
		  <manifest>
			%v
		  </manifest>
		  <spine toc="ncx">
			%v
		  </spine>
			%v
		</package>
	`
	if len(guide) > 0 {
		guide = `<guide>` + guide + `</guide>`
	}
	pkg = fmt.Sprintf(pkg, meta, manifest, spine, guide)
	if err := ioutil.WriteFile(basePath+"/content.opf", []byte(pkg), os.ModePerm); err != nil {
		panic(err)
	}
}

//最后一步
func ConvertToPdf() {

}
