package converter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"time"

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
			//设置默认值
			if len(cfg.Timestamp) == 0 {
				cfg.Timestamp = time.Now().Format("2006-01-02 15:04:05")
			}
			converter = &Converter{
				Config:   cfg,
				BasePath: basepath,
			}
		}
	}
	return
}

//执行文档转换
func (this *Converter) Convert() (err error) {
	//defer this.converterDefer() //最后移除创建的多余而文件

	this.generateMimeType()
	this.generateMetaInfo()
	this.generateTocNcx()     //生成目录
	this.generateTitlePage()  //生成封面
	this.generateContentOpf() //这个必须是generate*系列方法的最后一个调用

	//创建导出文件夹
	if err = os.Mkdir(output, os.ModePerm); err == nil {
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
								<image width="800" height="1068" xlink:href="` + strings.TrimPrefix(this.Config.Cover, "./") + `"/>
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
func (this *Converter) generateTocNcx() (err error) {
	ncx := `<?xml version='1.0' encoding='utf-8'?>
			<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1" xml:lang="%v">
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
	codes, _ := tocToXml(this.Config.Toc, 0, 1)
	ncx = fmt.Sprintf(ncx, this.Config.Language, this.Config.Title, strings.Join(codes, ""))
	return ioutil.WriteFile(this.BasePath+"/toc.ncx", []byte(ncx), os.ModePerm)
}

//生成content.opf文件
//倒数第二步调用
//TODO 这里需要优化和测试
func (this *Converter) generateContentOpf() (err error) {
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
	meta = fmt.Sprintf(meta, this.Config.Title, this.Config.Contributor, this.Config.Publisher, this.Config.Description, this.Config.Language, this.Config.Creator, this.Config.Timestamp)
	if len(this.Config.Cover) > 0 {
		meta = meta + `<meta name="cover" content="cover"/>`
		guide = `<reference href="titlepage.xhtml" title="Cover" type="cover"/>`
		manifest = fmt.Sprintf(`<item href="%v" id="cover" media-type="%v"/>`, this.Config.Cover, GetMediaType(filepath.Ext(this.Config.Cover)))
	}

	spine := ``
	//扫描所有文件
	if files, err := filetil.ScanFiles(this.BasePath); err == nil {
		manifestArr := []string{}
		spineArr := []string{}
		basePath := strings.Replace(this.BasePath, "\\", "/", -1)
		for _, file := range files {
			if !file.IsDir && file.Name != "book.json" {
				id := cryptil.Md5Crypt(file.Path)
				ext := strings.ToLower(filepath.Ext(file.Path))
				sourcefile := strings.TrimPrefix(file.Path, basePath+"/")
				mt := GetMediaType(ext)
				if sourcefile != strings.TrimLeft(this.Config.Cover, "./") && mt != "" { //不是封面图片，且media-type不为空
					manifestArr = append(manifestArr,
						fmt.Sprintf(`<item href="%v" id="%v" media-type="%v"/>`, sourcefile, id, mt),
					)
					if ext == ".html" || ext == ".xhtml" {
						spineArr = append(spineArr, fmt.Sprintf(`<itemref idref="%v"/>`, id))
					}
					if ext == ".ncx" {
						spineArr = append(spineArr, `<itemref idref="ncx"/>`)
					}
				}
			}
		}
		manifest = manifest + strings.Join(manifestArr, "\n")
		spine = strings.Join(spineArr, "\n")
	} else {
		return err
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
	return ioutil.WriteFile(this.BasePath+"/content.opf", []byte(pkg), os.ModePerm)
}

//最后一步
func ConvertToPdf() {

}
