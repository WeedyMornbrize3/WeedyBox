// internal/model/library.go
package model

import "time"

// LibraryWork 是「游戏库」里的一条记录：一次 DLsite 抓取的成果 + 本地缓存信息。
//
// 设计要点：
//   - RawJSON 存的是整份 DLsiteWork 的 json（原样），所以哪怕以后前端要新字段、
//     或者解析器加了字段，也不用改表结构 —— 老数据里有的字段都能读出来。
//   - Name / Maker / CoverURL 等列是为「列表页不解析 json 就能排序、筛选、显示」而冗余的，
//     它们与 RawJSON 同源，写入时一起写。
//   - Cover* 系列指向 preImg/ 下已经下载好的封面文件：再次访问时直接读本地文件，
//     不必重新联网（这是「优先使用缓存」的落点）。
type LibraryWork struct {
	// Source 是元数据来源（当前只有 "dlsite"）。表本身是来源无关的：
	// 作品信息放公共列，来源特有的字段全部留在 RawJSON 里。
	Source      string `json:"source"`
	WorkID      string `json:"workId"` // RJ01721474
	Name        string `json:"name"`
	Maker       string `json:"maker"`
	MakerID     string `json:"makerId,omitempty"`
	Series      string `json:"series,omitempty"`
	SeriesID    string `json:"seriesId,omitempty"`
	WorkType    string `json:"workType,omitempty"`
	AgeRating   string `json:"ageRating,omitempty"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	CoverURL    string `json:"coverUrl,omitempty"`
	SourceURL   string `json:"sourceUrl,omitempty"`
	RawJSON     string `json:"rawJson"`
	FetchedAt   string `json:"fetchedAt,omitempty"`
	// 下面这几个是本地缓存指针，没下载过封面时全为空
	CoverDir     string `json:"coverDir,omitempty"`
	CoverPath    string `json:"coverPath,omitempty"`
	CoverAbsPath string `json:"coverAbsPath,omitempty"`
	CoverBytes   int64  `json:"coverBytes"`
	CoverSuffix  string `json:"coverSuffix,omitempty"`
	// 入库时间与最后一次更新时间（由数据库维护）
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CoverFile 是一次封面下载的落地记录。
//
// 为什么要单独一张表：同一个作品会被反复点「下载封面」，而下载实现刻意
// **不覆盖**已有文件（重名加 -2 / -3 后缀）。只有把每一次的文件位置都记下来，
// 才能知道「这个作品到底在磁盘上有哪几张图、分别在哪个目录」。
type CoverFile struct {
	ID        int64     `json:"id"`
	Source    string    `json:"source"` // 元数据来源，同 LibraryWork.Source
	WorkID    string    `json:"workId"`
	Dir       string    `json:"dir"`     // 相对工作目录，如 preImg
	Path      string    `json:"path"`    // 相对路径，如 preImg/RJ01721474_cover.jpg
	AbsPath   string    `json:"absPath"` // 绝对路径，读图时用这个
	Bytes     int64     `json:"bytes"`
	Suffix    string    `json:"suffix"`
	CoverURL  string    `json:"coverUrl"`
	CreatedAt time.Time `json:"createdAt"`
}
