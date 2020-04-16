package v1

import (
	"math/rand"
	"time"

	"github.com/osu-datenshi/api/common"
)

var rn = rand.New(rand.NewSource(time.Now().UnixNano()))

var kaomojis = [...]string{
	"Σ(ノ°▽°)ノ",
	"( ƅ°ਉ°)ƅ",
	"ヽ(　･∀･)ﾉ",
	"˭̡̞(◞⁎˃ᆺ˂)◞*✰",
	"(p^-^)p",
	"(ﾉ^∇^)ﾉﾟ",
	"ヽ(〃･ω･)ﾉ",
	"(۶* ‘ꆚ’)۶”",
	"（。＞ω＜）。",
	"（ﾉ｡≧◇≦）ﾉ",
	"ヾ(｡･ω･)ｼ",
	"(ﾉ･д･)ﾉ",
	".+:｡(ﾉ･ω･)ﾉﾞ",
	"Σ(*ﾉ´>ω<｡`)ﾉ",
	"ヾ（〃＾∇＾）ﾉ♪",
	"＼（＠￣∇￣＠）／",
	"＼(^▽^＠)ノ",
	"ヾ(@^▽^@)ノ",
	"(((＼（＠v＠）／)))",
	"＼(*T▽T*)／",
	"＼（＾▽＾）／",
	"＼（Ｔ∇Ｔ）／",
	"ヽ( ★ω★)ノ",
	"ヽ(；▽；)ノ",
	"ヾ(。◕ฺ∀◕ฺ)ノ",
	"ヾ(＠† ▽ †＠）ノ",
	"ヾ(＠^∇^＠)ノ",
	"ヾ(＠^▽^＠)ﾉ",
	"ヾ（＠＾▽＾＠）ノ",
	"ヾ(＠゜▽゜＠）ノ",
	"(.=^・ェ・^=)",
	"((≡^⚲͜^≡))",
	"(^･o･^)ﾉ”",
	"(^._.^)ﾉ",
	"(^人^)",
	"(=；ェ；=)",
	"(=｀ω´=)",
	"(=｀ェ´=)",
	"（=´∇｀=）",
	"(=^･^=)",
	"(=^･ｪ･^=)",
	"(=^‥^=)",
	"(=ＴェＴ=)",
	"(=ｘェｘ=)",
	"＼(=^‥^)/’`",
	"~(=^‥^)/",
	"└(=^‥^=)┐",
	"ヾ(=ﾟ･ﾟ=)ﾉ",
	"ヽ(=^･ω･^=)丿",
	"d(=^･ω･^=)b",
	"o(^・x・^)o",
	"V(=^･ω･^=)v",
	"(⁎˃ᆺ˂)",
	"(,,^・⋏・^,,)",
}

var randomSentences = [...]string{
	"Datenshi? What is that?",
	"The brace is on fire!",
	"deverupa ga daisuki!",
	"It works!!!!",
	"Feelin' groovy!",
	"sudo rm -rf / (wait... did i forgot something here?)",
	"Momma, I don't wanna die",
	"Aoba haven't watch anime for 1 year",
	"I don't have money to buy new laptop",
	"This API is skidded :OMEGALUL:",
	"Relax server, yes.",
	"AWC hype Pepega",
	"Daisuki~",
	"Ainu Client is a big meme? nahh",
	":thonk:",
	"If you love me, then say so!",
	"Recommend me something.",
	"Hi, this is Aoba stuff",
	"Despacito 2019 v2",
	"ilyt, are you reading this?",
	"jackson... i'm sorry... for everything...",
	"Simon, u broke relax profile ~Aoba 2019",
	"All we had to do was follow the damn train, CJ.",
	"Free supporter for indo people!",
	"fake ass 5* map ~Aoffy 2019",
	"im bad at osu ~Aoba 2019",
	"WHAT ARE YOU DOING?!",
	"Are you reading this btw?",
	"donate me pls im poor",
}

func surpriseMe() string {
	n := int(time.Now().UnixNano())
	return randomSentences[n%len(randomSentences)] + " " + kaomojis[n%len(kaomojis)]
}

type pingResponse struct {
	common.ResponseBase
	ID              int                   `json:"user_id"`
	Privileges      common.Privileges     `json:"privileges"`
	UserPrivileges  common.UserPrivileges `json:"user_privileges"`
	PrivilegesS     string                `json:"privileges_string"`
	UserPrivilegesS string                `json:"user_privileges_string"`
}

// PingGET is a message to check with the API that we are logged in, and know what are our privileges.
func PingGET(md common.MethodData) common.CodeMessager {
	var r pingResponse
	r.Code = 200

	if md.ID() == 0 {
		r.Message = "You have not given us a token, so we don't know who you are! But you can still login with POST /tokens " + kaomojis[rn.Intn(len(kaomojis))]
	} else {
		r.Message = surpriseMe()
	}

	r.ID = md.ID()
	r.Privileges = md.User.TokenPrivileges
	r.UserPrivileges = md.User.UserPrivileges
	r.PrivilegesS = md.User.TokenPrivileges.String()
	r.UserPrivilegesS = md.User.UserPrivileges.String()

	return r
}

type surpriseMeResponse struct {
	common.ResponseBase
	Cats [100]string `json:"cats"`
}

// SurpriseMeGET generates cute cats.
//
// ... Yes.
func SurpriseMeGET(md common.MethodData) common.CodeMessager {
	var r surpriseMeResponse
	r.Code = 200
	for i := 0; i < 100; i++ {
		r.Cats[i] = surpriseMe()
	}
	return r
}
