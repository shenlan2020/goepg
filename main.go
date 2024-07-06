package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liuzl/gocc"
)

type Tv struct {
	Channel   []Channel   `xml:"channel"`
	Programme []Programme `xml:"programme"`
}

type Channel struct {
	ID          string      `xml:"id,attr"`
	DisplayName DisplayName `xml:"display-name"`
}

type DisplayName struct {
	Text string `xml:",chardata"`
}

type Programme struct {
	Channel string `xml:"channel,attr"`
	Start   string `xml:"start,attr"`
	Stop    string `xml:"stop,attr"`
	Title   Title  `xml:"title"`
	Desc    Desc   `xml:"desc"`
}

type Title struct {
	Text string `xml:",chardata"`
}

type Desc struct {
	Text string `xml:",chardata"`
}

var (
	cache       sync.Map
	cacheExpiry = 60 * time.Second
	fetchURL    = "https://epg.112114.xyz/pp.xml"
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
	epglists    = []string{"1905极限反转","1905环球经典","4K综艺","ABCAUSTRALIA","ALJAZEERA","AMC","ANIMAX","ARIRANGTV","AXN","BBCWORLDNEWS","BLUEANTENTERTAINMENT","BLUEANTEXTREME","BTV体育","BTV影视","BTV文艺","BTV新闻","BTV生活","BTV科教","BTV财经","CCTV1","CCTV10","CCTV11","CCTV12","CCTV13","CCTV14","CCTV15","CCTV16","CCTV17","CCTV2","CCTV3","CCTV4","CCTV4K","CCTV4欧洲","CCTV4美洲","CCTV5","CCTV5+","CCTV5PLUS","CCTV6","CCTV7","CCTV8","CCTV8K","CCTV9","CCTVNEWS","CCTV娱乐","CCTV怀旧剧场","CCTV戏曲","CCTV第一剧场","CCTV风云剧场","CDTV1","CDTV2","CDTV3","CDTV4","CDTV5","CDTV6","CETV1","CETV2","CETV3","CETV4","CGTN","CGTN俄语","CGTN新闻","CGTN法语","CGTN纪录","CGTN西语","CGTN阿语","CHC动作电影","CHC家庭影院","CHC影迷电影","CHC高清电影","CI","CINEMAWORLD","CINEMAX","CINEMAX热门影院","CMC北美","CMC香港","CNBCHONGKONG","CNEX","DISCOVERYASIA","DISCOVERY科学","DMAX","DONGFANG1","DW","ELTV","EURONEWS","FASHIONONE","FRANCE24","GOODTV","GOODTV2","GTV游戏竞技","HBO","HBOFAMILY","HBOHITS","HBOSIGNATURE","HBO原创巨献","HBO强档巨献","HBO溫馨家庭","HISTORY","HITS","HKS","HOY76","HOY77","HOY78","HUBEI7","HZTV1","HZTV3","IFUN1","IFUN3","IPTV少儿动画","IPTV热播剧场","IPTV经典电影","IPTV谍战剧场","IPTV魅力时尚","J2","LIFETIME","LSTIME电影台","LUXETV","MEZZOLIVEHD","MYCINEMAEUROPE","NBTV1","NBTV2","NBTV3","NBTV4","NBTV5","NHKWORLD","NHKWORLDPREMIUM","NICKELODEON","NICKJR.","NOW剧集台","NOW华剧台","NOW新闻台","NOW爆谷台","NOW爆谷星影台","NOW直播台","NOW财经台","OUTDOOR","QHTV1","QZTV1","QZTV2","RTHK31","RTHK32","RTHK33","RTHK34","RTHK35","SBN全球财经台","SCTV1","SCTV2","SCTV3","SCTV4","SCTV5","SCTV7","SCTV8","SCTV9","SITV14","SKYNEWS","SONYMAX","SONYSAB","TFC","TLC旅游生活","TRACESPORTS","TRACEURBAN","TRAVELCHANNEL","TV5MONDE","TVB PLUS","TVBS新闻台","TVBS欢乐台","TVB剧场","TVB经典台","TVN","VIUTV","VIUTVSIX","WZTV1","七彩戏剧","三沙卫视","三立综合台","上海","上海外语","上海教育","上海教育台","上海新闻综合","上海星尚","上海第一财经","上海纪实","上海纪实人文","上海艺术人文","上海都市","上视东方影视","上视纪实","世界地理","东北热剧","东南卫视","东方卫视","东方影视","东方电影","东方财经","东森亚洲卫视","东森亚洲新闻台","东莞新闻","东莞生活资讯","中华特产","中华美食","中国交通","中国功夫","中国天气","中国教育1台","中国教育2台","中国教育4台","中国教育一套","中天亚洲台","中天新闻台","中央台–珠江","中学生","中山公共","中山教育","中山综合","中视","中视新闻","中视经典","中视菁采台","之江纪录","乌鲁木齐都市","乐山新闻综合","乐游","书画","云上电影院","云南公共","云南卫视","云南娱乐","云南少儿","云南康旅","云南影视","云南生活","云南都市","云浮综合","五星体育","亚洲新闻台","仙游电视","优优宝贝","优漫卡通","佛山公共","佛山南海","佛山影视","佛山综合","佛山顺德","先锋乒羽","全纪实","六安公共","六安新闻综合","六安都市生活","兰州公共","兰州文旅","兰州新闻综合","兰州生活经济","兵器科技","兵团卫视","内蒙古农牧","内蒙古卫视","内蒙古少儿","内蒙古经济生活","内蒙古综合","军事评论","军旅剧场","农业致富","农林卫视","冬奥纪实","凤凰中文","凤凰卫视中文台","凤凰卫视资讯台","凤凰卫视香港台","凤凰资讯","凤凰香港","创世电视","力量影院","动作电影","动漫秀场","动物星球","劲爆体育","北京4K","北京4K电影","北京体育休闲","北京卫视","北京国际","北京国际频道","北京影视","北京文艺","北京新闻","北京淘BABY","北京淘剧场","北京淘娱乐","北京淘电影","北京淘精彩","北京生活","北京纪实","北京纪实科教","北京财经","北海公共","北海新闻综合","北海经济科教","半岛英语","华视","南京信息","南京十八","南京娱乐","南京少儿","南京影视","南京教科","南京新闻综合","南京生活","南京电视台","南充科教","南充综合","南方卫视","南方少儿","南方影视","南方经济","南方综艺","南昌公共","南昌新闻综合","南昌资讯","南昌都市","南通新闻综合","博斯无限台","博斯网球台","博斯运动一台","博斯高球1台","卡酷动画","卡酷少儿","卫生健康","厦门一套","厦门三套","厦门二套","厦门卫视","厦门海峡","厦门移动","厦门综合","发现之旅","古装剧场","台州公共","台州城市生活","台州文化生活","台州新闻综合","台视","吉林乡村","吉林公共新闻","吉林卫视","吉林市公共","吉林市新闻","吉林市科教","吉林影视","吉林生活","吉林综艺文化","吉林都市","吉视乡村","吉视影视","吉视生活","吉视综艺文化","吉视都市","吴江新闻综合","呼和浩特影视","呼和浩特综合","呼和浩特都市","咪咕24小时体育","咪咕NBA-1","咪咕综合体育","咪咕足球","哈哈炫动","哒啵电竞","哒啵赛事","嘉佳卡通","四川乡村","四川卫视","四川妇女儿童","四川影视文艺","四川文化旅游","四川新闻","四川科教","四川经济","四海钓鱼","国会频道1","国会频道2","国学","国家地理高清","大庆新闻综合","大湾区卫视","大湾区卫视海外版","天元围棋","天映CM+","天映印度尼西亚","天映新加坡","天映经典","天映马来西亚","天津体育","天津卫视","天津少儿","天津影视","天津教育","天津文艺","天津新闻","天津都市","天龙八部集","太原影视","太原教育","太原文体","太原新闻综合","太原百姓","太原社教法制","央视台球","央视精品","女性时尚","宁夏公共","宁夏卫视","宁夏少儿","宁夏教育","宁夏文旅","宁夏经济","安多卫视","安徽公共","安徽农业科教","安徽卫视","安徽国际","安徽影视","安徽科教","安徽经济生活","安徽经视","安徽综艺","安徽综艺体育","宜宾新闻综合","宜春新闻综合","客家生活","客家电视台","家庭剧场","寰宇新闻","小小课堂","山东体育","山东体育休闲","山东公共","山东农科","山东卫视","山东少儿","山东影视","山东教育","山东教育卫视","山东文旅","山东新闻","山东生活","山东综艺","山东齐鲁","山西公共","山西卫视","山西影视","山西文体生活","山西社会与法治","山西经济","山西经济与科技","岭南戏曲","峨嵋电影","峨眉电影","崇左综合","常州公共","常州新闻","常州生活","常州综合","常州都市","幸福空间居家台","广东体育","广东公共","广东卫视","广东国际","广东少儿","广东影视","广东新闻","广东民生","广东现代教育","广东珠江","广东移动","广东经济","广东经济科教","广东综艺","广元公共","广元新闻综合","广州影视","广州新闻","广州法治","广州竞赛","广州综合台","广西卫视","广西国际","广西影视","广西新闻","广西综艺","广西综艺旅游","广西都市","康巴卫视","延边1台","延边2台","延边卫视","弈坛春秋","彩民在线","影迷数位电影","影迷数位纪实","徐州文艺影视","徐州新闻综合","徐州经济生活","快乐垂钓","怀旧剧场","怡伴健康","惊悚悬疑","成都公共","成都少儿","成都影视文艺","成都新闻综合","成都经济资讯","成都都市生活","成龙作品集","揭阳生活","揭阳综合","摄影","攀枝花新闻综合","收藏天下","文化精品","文物宝库","新余公共","新余新闻综合","新动漫","新片放映厅","新疆体育健康","新疆卫视","新疆少儿","新疆汉语影视","新疆汉语经济","新疆汉语综艺","新科动漫","新视觉","旅游卫视","无线新闻","无锡娱乐","无锡新闻综合","无锡生活","无锡电视娱乐","无锡电视生活","无锡电视经济","无锡经济","无锡都市资讯","日照新闻综合","日照科教","早期教育","明星大片","明珠台","星光影院","星空卫视","星空购物","智林体育","曼联电视","来宾综合","杭州导视","杭州少儿","杭州影视","杭州房产","杭州文化","杭州生活","杭州综合","杭州西湖明珠","杭州青少体育","松原综合","柳州公共","柳州新闻综合","柳州科教","梅州-1","梨园","欢乐剧场","欢笑剧场","歌手2024","武术世界","武汉外语","武汉少儿","武汉教育","武汉文体","武汉新闻综合","武汉电视剧","武汉科教生活","武汉经济","民视","民视台湾台","民视第一台","民视综艺台","求索动物","求索生活","求索科学","求索纪录","求索记录","汕头文旅体育","汕头新闻综合","汕头经济生活","汕头综合","汕尾文化生活","汕尾新闻综合","江苏休闲体育","江苏优漫卡通","江苏体育休闲","江苏卫视","江苏国际","江苏城市","江苏影视","江苏教育","江苏教育电视台","江苏新闻","江苏综艺","江西公共","江西公共农业","江西卫视","江西少儿","江西影视","江西影视旅游","江西新闻","江西移动","江西经济","江西经济生活","江西都市","江门侨乡生活","江门综合","汽摩","河北公共","河北农民","河北卫视","河北少儿科教","河北影视","河北经济","河北都市","河南乡村","河南公共","河南卫视","河南新农村","河南新闻","河南梨园","河南民生","河南法治","河南电视剧","河南都市","河源公共","泉州新闻","法治天地","泰州一套","泰州三套","泸州新闻综合","济南综合","济南鲁中","浙江NEWS","浙江公共新闻","浙江卫视","浙江国际","浙江少儿","浙江影视","浙江教科影视","浙江教育","浙江数码时代","浙江新闻","浙江民生","浙江民生休闲","浙江经济生活","浙江经视","浙江钱江都市","海南公共","海南卫视","海南少儿","海南文旅","海南新闻","海南自贸","海峡卫视","淮南公共","淮南新闻综合","深圳体育健康","深圳公共","深圳卫视","深圳国际","深圳娱乐","深圳少儿","深圳电视剧","深圳移动","深圳财经","深圳财经生活","深圳都市","深视体育健康","清远公共","清远新闻综合","游戏竞技","游戏风云","湖北公共新闻","湖北卫视","湖北垄上","湖北影视","湖北教育","湖北生活","湖北经视","湖北综合","湖南卫视","湖南卫视国际","湖南国际","湖南娱乐","湖南教育","湖南爱晚","湖南电影","湖南电视剧","湖南经视","湖南都市","湛江公共","湛江综合","滨州民生","滨州综合","漳州新闻","潮妈辣婆","潮安综合","潮州民生","潮州综合","澜湄国际","澳亚卫视","澳视体育","澳视卫星","澳视澳门","澳视综艺","澳视葡文","澳视资讯","炫舞未来","热播精选","爱大剧","爱尔达娱乐台","爱情喜剧","玉林公共","玉林新闻综合","环球奇观","环球旅游","环球经典","现代女性","现代教育","珠江","珠海新闻","珠海生活","甘肃公共","甘肃卫视","甘肃少儿","甘肃文化影视","甘肃经济","甘肃都市","生态环境","生活时尚","电竞天堂","电视指南","留学世界","百姓健康","百色综合","眉山综合","睛彩中原","睛彩广场舞","睛彩竞技","睛彩篮球","睛彩羽毛球","石家庄娱乐","石家庄新闻综合","石家庄生活","石家庄都市","福建体育","福建公共","福建少儿","福建教育","福建文体","福建新闻","福建旅游","福建电视剧","福建经济","福建经视","福建综合","移动戏曲","第一剧场","第一财经","篮球","精品体育","精品大剧","精品纪录","精品萌宠","精彩影视","红色轮播","纪实人文","纪实科教","纬来体育台","纬来戏剧台","纬来日本台","纬来电影台","纬来精彩台","纬来综合台","纬来音乐台","纯享4K","经典剧场","经济科教","绵阳影视科技","绵阳新闻综合","置业","美亚电影","美亚高清电影台","翡翠台","老故事","肇庆新闻","肇庆生活服务","自贡公共","自贡综合","苏州文化生活","苏州文化生活-苏州新闻综合-苏州生活资讯-苏州社会经济","苏州新闻综合","苏州生活资讯","苏州社会经济","茂名公共","茂名综合","茶","莆田一套","莆田二套","莲花卫视","藏语卫视","西宁生活","西宁综合","西安丝路","西安商务资讯","西安商务资迅","西安影视","西安新闻综合","西安移动电视","西安都市","西湖明珠","西藏卫视","西藏影视","西藏影视文化","西藏藏语卫视","证券资讯","象视界","财富天下","贵州乡村生态","贵州公共","贵州卫视","贵州大众生活","贵州影视","贵州影视文艺","贵州生活","贵州科教","贵州科教健康","贵阳生活","贵阳综合","赣州公共","赣州教育","赣州新闻综合","超级体育","超级电影","超级电视剧","超级综艺","车迷","辽宁体育休闲","辽宁公共","辽宁北方","辽宁卫视","辽宁影视剧","辽宁教育青少","辽宁生活","辽宁经济","辽宁都市","达文西","追剧少女","遂宁公共","遂宁综合","郑州商都","郑州妇女","郑州戏曲","郑州文体","郑州新闻","郑州生活","都市剧场","采昌影剧","重广融媒","重庆农村","重庆卫视","重庆国际","重庆娱乐","重庆少儿","重庆影视","重庆文体娱乐","重庆新农村","重庆新闻","重庆时尚生活","重庆生活资讯","重庆社会与法","重庆科教","重庆移动","重温经典","金华新闻","金牌综艺","金色学堂","金鹰卡通","金鹰纪实","钦州公共","钦州综合","钱江","钱江都市","银川公共","银川文体","银川生活","锦州公共","锦州新闻","锦州都市","长春市民","长春文旅体育","长春汽车","长春经济","长春综合","长沙影视","长沙政法","防城港公共","防城港新闻综合","陕西一套","陕西七套","陕西三套","陕西二套","陕西五套","陕西体育休闲","陕西八套","陕西公共","陕西六套","陕西农林卫视","陕西卫视","陕西四套","陕西影视","陕西新闻资讯","陕西生活","陕西都市青春","青岛tv1","青岛tv2","青岛tv3","青岛tv4","青岛tv5","青岛tv6","青岛影视","青岛教育","青岛新闻综合","青岛生活服务","青岛财经资讯","青岛都市","青海卫视","青海经视","青海都市","靖天卡通台","靖天戏剧台","靖天日本台","靖天映画","靖天欢乐台","靖天电影台","靖天综合台","靖天育乐台","靖天资讯台","靖洋卡通台","靖洋戏剧台","鞍山新闻综合","韩国娱乐台KMTV","韶关新闻","风云剧场","风云足球","风云音乐","香港卫视","高尔夫网球","高清大片","魅力足球","黄河卫视","黑莓动画","黑莓电影","黑龙江农业科教","黑龙江卫视","黑龙江少儿","黑龙江影视","黑龙江文体","黑龙江新闻","黑龙江新闻法制","黑龙江都市","齐鲁","龙华戏剧","龙华日韩台","龙华电影","龙岩综合","龙祥时代"}
	converter   *gocc.OpenCC
)

func init() {
	var err error
	converter, err = gocc.New("t2s")
	if err != nil {
		fmt.Println("Error initializing converter:", err)
	}
}

func fetchEPGData() {
	for {
		req, err := http.NewRequest("GET", fetchURL, nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			continue
		}

		req.Header.Set("User-Agent", userAgent)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error fetching data:", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Println("Error: unable to fetch XML data")
			continue
		}

		xmlData, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			continue
		}

		var tv Tv
		err = xml.Unmarshal(xmlData, &tv)
		if err != nil {
			fmt.Println("Error unmarshalling XML:", err)
			continue
		}

		cache.Store("epg", tv)
		fmt.Println("EPG data updated")

		time.AfterFunc(cacheExpiry, fetchEPGData)
		break
	}
}

func formatDateTime(timeStr string) (string, string) {
	if strings.Contains(timeStr, "-") {
		return timeStr, ""
	}

	if len(timeStr) < 8 {
		return "", ""
	}

	year := timeStr[:4]
	month := timeStr[4:6]
	day := timeStr[6:8]
	date := fmt.Sprintf("%s-%s-%s", year, month, day)

	var time string
	if len(timeStr) >= 12 {
		hour := timeStr[8:10]
		minute := timeStr[10:12]
		time = fmt.Sprintf("%s:%s", hour, minute)
	}

	return date, time
}

func getCurrentDateInBeijing() string {
	TimeLocation, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		TimeLocation = time.FixedZone("CST", 8*60*60)
	}
	currentTime := time.Now().In(TimeLocation)
	return currentTime.Format("2006-01-02")
}

func generateDefaultEPG() []map[string]string {
	epgData := make([]map[string]string, 24)
	for hour := 0; hour < 24; hour++ {
		startTime := fmt.Sprintf("%02d:00", hour)
		endTime := fmt.Sprintf("%02d:00", (hour+1)%24)
		epgData[hour] = map[string]string{
			"start": startTime,
			"end":   endTime,
			"title": "精彩节目-暂未提供节目预告信息",
			"desc":  "",
		}
	}
	return epgData
}

func sanitizeChannelName(channel string) string {
	tid := strings.ToUpper(channel)
	re := regexp.MustCompile(`\[.*?\]|[0-9\.]+M|[0-9]{3,4}[pP]|[0-9\.]+FPS`)
	tid = re.ReplaceAllString(tid, "")
	tid = strings.TrimSpace(tid)
	re = regexp.MustCompile(`超清|高清$|蓝光|频道$|标清|FHD|HD$|HEVC|HDR|-|\s+`)
	tid = re.ReplaceAllString(tid, "")
	tid = strings.TrimSpace(tid)

	if strings.Contains(tid, "CCTV") && !strings.Contains(tid, "CCTV4K") {
		re := regexp.MustCompile(`CCTV[0-9+]{1,2}[48]?K?`)
		matches := re.FindStringSubmatch(tid)
		if len(matches) > 0 {
			tid = strings.Replace(matches[0], "4K", "", -1)
		} else {
			re = regexp.MustCompile(`CCTV[^0-9]+`)
			matches = re.FindStringSubmatch(tid)
			if len(matches) > 0 {
				tid = strings.Replace(matches[0], "CCTV", "", -1)
			}
		}
	} else {
		tid = strings.Replace(tid, "BTV", "北京", -1)
	}
	return tid
}

func getMatchedChannel(query string, tv Tv, date string) string {
	normalizedQuery := sanitizeChannelName(query)
	simplifiedQuery, err := converter.Convert(normalizedQuery)
	if err != nil {
		simplifiedQuery = normalizedQuery
	}

	priorityMatch := ""
	secondaryMatch := ""
	matched := ""

	for _, epg := range epglists {
		upperEPG := strings.ToUpper(epg)
		if strings.HasPrefix(upperEPG, normalizedQuery) || strings.HasPrefix(upperEPG, simplifiedQuery) {
			if hasEPGData(epg, tv, date) {
				return epg
			}
		} else if strings.Contains(upperEPG, normalizedQuery) || strings.Contains(upperEPG, simplifiedQuery) {
			if isChinesePrefix(epg) {
				if priorityMatch == "" {
					priorityMatch = epg
				}
			} else {
				if secondaryMatch == "" {
					secondaryMatch = epg
				}
			}
		}
	}

	if priorityMatch != "" && hasEPGData(priorityMatch, tv, date) {
		return priorityMatch
	}
	if secondaryMatch != "" && hasEPGData(secondaryMatch, tv, date) {
		return secondaryMatch
	}

	for i := len(normalizedQuery); i > 0; i-- {
		subQuery := normalizedQuery[:i]
		for _, epg := range epglists {
			upperEPG := strings.ToUpper(epg)
			if strings.HasPrefix(upperEPG, subQuery) {
				if hasEPGData(epg, tv, date) {
					if len(epg) > len(matched) {
						matched = epg
					}
				}
			}
		}
	}

	if matched != "" {
		return matched
	}

	return "未知频道"
}

func isChinesePrefix(s string) bool {
	re := regexp.MustCompile(`^[\p{Han}]`)
	return re.MatchString(s)
}

func hasEPGData(channel string, tv Tv, date string) bool {
	for _, programme := range tv.Programme {
		if strings.Contains(strings.ToUpper(programme.Channel), strings.ToUpper(channel)) && strings.HasPrefix(programme.Start, strings.ReplaceAll(date, "-", "")) {
			if programme.Title.Text != "" {
				return true
			}
		}
	}
	return false
}

func handleEPG(c *gin.Context) {
	channel := strings.ToUpper(c.DefaultQuery("ch", "CCTV1"))
	dateParam := c.DefaultQuery("date", getCurrentDateInBeijing())
	date, _ := formatDateTime(dateParam)
	epgInterface, exists := cache.Load("epg")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "EPG data not available"})
		return
	}
	tv := epgInterface.(Tv)

	channel = getMatchedChannel(channel, tv, date)

	var epgData []map[string]string
	for _, programme := range tv.Programme {
		if programme.Channel == channel && strings.HasPrefix(programme.Start, strings.ReplaceAll(date, "-", "")) {
			_, startTime := formatDateTime(programme.Start)
			_, endTime := formatDateTime(programme.Stop)
			epgData = append(epgData, map[string]string{
				"start": startTime,
				"end":   endTime,
				"title": programme.Title.Text,
				"desc":  programme.Desc.Text,
			})
		}
	}

	if len(epgData) == 0 {
		epgData = generateDefaultEPG()
	}

	response := map[string]any{
		"date":         date,
		"channel_name": channel,
		"epg_data":     epgData,
	}

	c.JSON(http.StatusOK, response)
}

func main() {
	go fetchEPGData()
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/json", handleEPG)
	r.Run(":27100")
}
