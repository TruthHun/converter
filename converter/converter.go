package converter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"time"

	"os/exec"

	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/gotil/filetil"
	"github.com/TruthHun/gotil/ziptil"
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
	FontSize    string   `json:"font_size"`    //默认的pdf导出字体大小
	PaperSize   string   `json:"paper_size"`   //页面大小
	MoreOptions []string `json:"more_options"` //更多导出选项
	Toc         []Toc    `json:"toc"`
}

var (
	output       = "output" //文档导出文件夹
	ebookConvert = "ebook-convert"
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
	defer this.converterDefer() //最后移除创建的多余而文件

	this.generateMimeType()
	this.generateMetaInfo()
	this.generateTocNcx()     //生成目录
	this.generateTitlePage()  //生成封面
	this.generateContentOpf() //这个必须是generate*系列方法的最后一个调用

	//将当前文件夹下的所有文件压缩成zip包，然后直接改名成content.epub
	f := this.BasePath + "/content.epub"
	os.Remove(f) //如果原文件存在了，则删除;
	if err = ziptil.Zip(f, this.BasePath); err == nil {
		//创建导出文件夹
		os.Mkdir(this.BasePath+"/"+output, os.ModePerm)
		if len(this.Config.Format) > 0 {
			for _, v := range this.Config.Format {
				fmt.Println("convert to " + v)
				switch strings.ToLower(v) {
				case "epub":
					err = this.convertToEpub()
				case "mobi":
					err = this.convertToMobi()
				case "pdf":
					err = this.convertToPdf()
				}
			}
		} else {
			err = this.convertToPdf()
		}
	}
	return
}

//删除生成导出文档而创建的文件
func (this *Converter) converterDefer() {
	//删除不必要的文件
	os.RemoveAll(this.BasePath + "/META-INF")
	os.RemoveAll(this.BasePath + "/content.epub")
	os.RemoveAll(this.BasePath + "/mimetype")
	os.RemoveAll(this.BasePath + "/toc.ncx")
	os.RemoveAll(this.BasePath + "/content.opf")
	os.RemoveAll(this.BasePath + "/titlepage.xhtml") //封面图片待优化
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
	os.MkdirAll(folder, os.ModePerm)
	err = ioutil.WriteFile(folder+"/container.xml", []byte(xml), os.ModePerm)
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
func (this *Converter) generateContentOpf() (err error) {
	var (
		guide       string
		manifest    string
		manifestArr []string
		spine       string //注意：如果存在封面，则需要把封面放在第一个位置
		spineArr    []string
	)

	meta := `<dc:title>%v</dc:title>
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
		spineArr = append(spineArr, `<itemref idref="titlepage"/>`)
	}

	//扫描所有文件
	if files, err := filetil.ScanFiles(this.BasePath); err == nil {
		basePath := strings.Replace(this.BasePath, "\\", "/", -1)
		for _, file := range files {
			if !file.IsDir {
				ext := strings.ToLower(filepath.Ext(file.Path))
				id := "ncx"
				if ext != ".ncx" {
					if file.Name == "titlepage.xhtml" {
						id = "titlepage"
					} else {
						id = cryptil.Md5Crypt(file.Path)
					}

				}
				sourcefile := strings.TrimPrefix(file.Path, basePath+"/")
				if mt := GetMediaType(ext); mt != "" { //不是封面图片，且media-type不为空
					if (ext == ".html" || ext == ".xhtml") && file.Name != "titlepage.xhtml" {
						spineArr = append(spineArr, fmt.Sprintf(`<itemref idref="%v"/>`, id))
					}
					if sourcefile != strings.TrimLeft(this.Config.Cover, "./") { //不是封面图片，则追加进来。封面图片前面已经追加进来了
						manifestArr = append(manifestArr, fmt.Sprintf(`<item href="%v" id="%v" media-type="%v"/>`, sourcefile, id, mt))
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

//转成epub
func (this *Converter) convertToEpub() (err error) {
	args := []string{
		this.BasePath + "/content.epub",
		this.BasePath + "/" + output + "/book.epub",
	}
	return exec.Command(ebookConvert, args...).Run()
}

//转成mobi
func (this *Converter) convertToMobi() (err error) {
	args := []string{
		this.BasePath + "/content.epub",
		this.BasePath + "/" + output + "/book.mobi",
	}
	return exec.Command(ebookConvert, args...).Run()
}

//转成pdf
func (this *Converter) convertToPdf() (err error) {
	args := []string{
		this.BasePath + "/content.epub",
		this.BasePath + "/" + output + "/book.pdf",
	}
	//页面大小
	if len(this.Config.PaperSize) > 0 {
		args = append(args, "--paper-size", this.Config.PaperSize)
	}
	//文字大小
	if len(this.Config.FontSize) > 0 {
		args = append(args, "--pdf-default-font-size", this.Config.FontSize)
	}

	//header template
	if len(this.Config.Header) > 0 {
		args = append(args, "--pdf-header-template", this.Config.Header)
	}

	//footer template
	if len(this.Config.Footer) > 0 {
		args = append(args, "--pdf-footer-template", this.Config.Footer)
	}

	return exec.Command(ebookConvert, args...).Run()
}

//最后一步
func ConvertToPdf() {

}
