package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Yaku struct {
	Name string
	Score int
}

type Word struct {
	Text string //単語の文字列
	Runes []rune //構成文字（サジェスト用）
	RegisteredBy string //登録したプレイヤー名
}

type WordYaku struct {
	Word string
	YakuTags []string //この単語に付けられた役のタグ（複数可）
	Score int //その時付けられた得点
	RegisteredAt string //登録日時
}

type Model struct {
	Player1 *Player
	Player2 *Player
	Yama []rune //山札（50音×4セット）
	WordDic map[string]*Word //有効なワード辞書（プレイヤーが登録）
	YakuDic map[string]*Yaku //役名→役情報
	WordYakuDic []WordYaku //単語と役の紐付け履歴（学習データ）
	CurrentPlayer int //1 or 2
	Phase string //ツモ、捨て牌、グループ作成、上がり判定など
}

type Player struct {
	Name string
	Tehai *Tehai
	Kawa []rune //このプレイヤーの捨て牌
	Score int
	RichiFlag bool //リーチ的な何か用
}

type Tehai struct {
	Bara []rune //未グループ化の牌
	Groups []Group //グループ化された単語
}

type Group struct {
	Word string //グループ化された文字列
	Runes []rune //構成文字
	IsNaki bool //鳴き（相手の捨て牌から取った）
	NakiFrom string //誰から鳴いたか（プレイヤー名）
}

// 山札を初期化する関数
func createYama() []rune {
	// 50音（清音のみ）
	gojuon := []rune{
		'あ', 'い', 'う', 'え', 'お',
		'か', 'き', 'く', 'け', 'こ',
		'さ', 'し', 'す', 'せ', 'そ',
		'た', 'ち', 'つ', 'て', 'と',
		'な', 'に', 'ぬ', 'ね', 'の',
		'は', 'ひ', 'ふ', 'へ', 'ほ',
		'ま', 'み', 'む', 'め', 'も',
		'や', 'ゆ', 'よ',
		'ら', 'り', 'る', 'れ', 'ろ',
		'わ', 'を', 'ん',
	}

	yama := []rune{}

	// 50音を4セット追加
	for i := 0; i < 4; i++ {
		yama = append(yama, gojuon...)
	}

	// 濁点（゛）を10枚
	for i := 0; i < 10; i++ {
		yama = append(yama, '゛')
	}

	// 半濁点（゜）を10枚
	for i := 0; i < 10; i++ {
		yama = append(yama, '゜')
	}

	// 伸ばし棒（ー）を10枚
	for i := 0; i < 10; i++ {
		yama = append(yama, 'ー')
	}

	// シャッフル
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(yama), func(i, j int) {
		yama[i], yama[j] = yama[j], yama[i]
	})

	return yama
}

// 学習データを保存する構造体
type GameData struct {
	WordDic map[string]*Word `json:"word_dic"`
	YakuDic map[string]*Yaku `json:"yaku_dic"`
	WordYakuDic []WordYaku `json:"word_yaku_dic"`
}

// 学習データをファイルに保存
func saveGameData(data *GameData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // 読みやすいように整形
	return encoder.Encode(data)
}

// 学習データをファイルから読み込み
func loadGameData(filename string) (*GameData, error) {
	file, err := os.Open(filename)
	if err != nil {
		// ファイルが存在しない場合は空のデータを返す
		if os.IsNotExist(err) {
			return &GameData{
				WordDic: make(map[string]*Word),
				YakuDic: make(map[string]*Yaku),
				WordYakuDic: []WordYaku{},
			}, nil
		}
		return nil, err
	}
	defer file.Close()

	var data GameData
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}
