package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	//"strconv"
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
	Turn int
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


func main() {

	p := tea.NewProgram(initialModel())
	if _,err := p.Run(); err != nil {
		fmt.Printf("error &v", err)
	}
}

func initialModel() Model {
	player1 := Player {
		Name: "Player1",
        Tehai: &Tehai{
            Bara: []rune{},
            Groups: []Group{},
        },
		Kawa: []rune{},
		Score: 0,
		RichiFlag: false,
	}

		player2 := Player {
		Name: "Player2",
        Tehai: &Tehai{
            Bara: []rune{},
            Groups: []Group{},
        },
		Kawa: []rune{},
		Score: 0,
		RichiFlag: false,
	}

	yama := createYama()

//	m.haipai()


	m := Model{
		Player1: &player1,
		Player2: &player2,
		Yama: yama,
		WordDic: make(map[string]*Word),
		YakuDic: make(map[string]*Yaku),
		WordYakuDic: []WordYaku{},
		CurrentPlayer: 1,
		Turn: 1,
		Phase: "ツモ",
		}

		return m

	}


func (m *Model) haipai() Model {
	for i := 0; i < 14; i++ {
		m.Player1.Tehai.Bara = append(m.Player1.Tehai.Bara, m.Yama[i])
		//m.Yamaからindexの要素を削除する。本当は直接YamaからTehai.Baraに移動できたらてっとり早いが
	}
	for i := 0; i < 14; i++ {
		m.Player2.Tehai.Bara = append(m.Player2.Tehai.Bara, m.Yama[i])
	    //Playersにしてなかったのでイテレート出来ないｗ
	}


	m.Yama = m.Yama[28:]
	return *m
}


/*
func (m Model) View() string {
	var s strings.Builder

	s.WriteString(playerStyle.Render("Playser2 information\n"))
	s.WriteString(textStyle.Render("Playser2 river"\n))
	s.WriteString("\n\n\n")

	s.WriteString(textStyle.Render("　　　　Turn:%s\n", m.Turn, m.Phase))
	s.WriteString(textStyle.Render("　　　　Phase:%s\n"))
	s.WriteString("\n")
	//メニュー表示枠
	s.WriteString("\n\n\n")

	kawa := ""
	for _,r := range m.Player1.Tehai.Bara {//runes
		kawa = append(kawa, r+"　")//Runeに全角空白文字列を足して表示したい
	}
	for _,g := range m.Player1.Tehai.Groups {
		kawa = append(kawa, g.Word+"　")
	}
	s.WriteString(textStyle.Render(kawa)
	s.WriteString(textStyle.Render("Player1 information\n"))

}

*/

func (m Model) View() string {
    var s strings.Builder
    
    s.WriteString(playerStyle.Render("Player2 information\n")) // 誤: Playser2 → 正: Player2
    s.WriteString(textStyle.Render("Player2 river\n")) // 誤: Playser2 → 正: Player2、閉じ引用符の位置が間違っている
    s.WriteString("\n\n\n")
    
    // 誤: Render()の引数が足りない。フォーマット文字列に%sが2つあるのに引数が1つしかない
    // 正: 引数を2つ渡すか、フォーマット文字列を修正する
    s.WriteString(textStyle.Render(fmt.Sprintf("　　　　Turn:%d\n", m.Turn)))
    s.WriteString(textStyle.Render(fmt.Sprintf("　　　　Phase:%s\n", m.Phase)))
    s.WriteString("\n")
    
    // メニュー表示枠
    s.WriteString("\n\n\n")
    
    // 誤: kawaはstring型なのにappendを使っている（appendはスライス用）
    // 正: 文字列連結には += または strings.Builder を使う
    kawa := ""
    for _, r := range m.Player1.Tehai.Bara { // rはrune型
        // runeを文字列に変換してから全角スペースを追加
        kawa += string(r) + "　"
    }
    
    for _, g := range m.Player1.Tehai.Groups {
        // g.Wordは既に文字列なので、そのまま全角スペースを追加
        kawa += g.Word + "　"
    }
    

    s.WriteString(textStyle.Render(kawa)) // kawaの内容を表示
	s.WriteString("\n")
	s.WriteString(playerStyle.Render("Player1 information\n"))
    
    return s.String() // 最後にstring()で文字列を返す必要がある
}


var (
	textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000"))
	playerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Background(lipgloss.Color("#000000"))
	//waterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	//mtStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B4513"))
)


/*
import (
    "github.com/charmbracelet/lipgloss"
)

// 基本的な枠付きスタイルの作り方
var boxStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()). // 角丸の枠
    BorderForeground(lipgloss.Color("255")). // 枠の色（白）
    Padding(1, 2) // 内側の余白（上下1、左右2）

// 使い方
func (m Model) View() string {
    // 枠で囲んで表示
    return boxStyle.Render("Hello, World!")
}

// 他の枠のスタイル例
var normalBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder()) // 通常の枠 ┌─┐│ │└─┘

var thickBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.ThickBorder()) // 太い枠 ┏━┓┃ ┃┗━┛

var doubleBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.DoubleBorder()) // 二重線の枠 ╔═╗║ ║╚═╝

// 枠の色を変える
var coloredBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("86")) // 水色の枠

// 背景色も付ける
var fancyBoxStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("255")). // 白い枠
    Background(lipgloss.Color("235")). // 暗い背景
    Foreground(lipgloss.Color("86")). // 文字色
    Padding(1, 2). // 内側の余白
    Margin(1) // 外側の余白


	
// あなたのコードに適用する例
func (m Model) View() string {
    var s strings.Builder
    
    // Player2の情報を枠で囲む
    player2Box := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("255")).
        Padding(0, 1)
    
    s.WriteString(player2Box.Render("Player2 information\nPlayer2 river"))
    s.WriteString("\n\n")
    
    // ターン情報を枠で囲む
    turnBox := lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color("86"))
    
    turnInfo := fmt.Sprintf("Turn:%s\nPhase:%s", m.Turn, m.Phase)
    s.WriteString(turnBox.Render(turnInfo))
    
    // 手牌を枠で囲む
    tehaiBorder := lipgloss.NewStyle().
        Border(lipgloss.DoubleBorder()).
        BorderForeground(lipgloss.Color("255")).
        Padding(1, 2)
    
    kawa := ""
    for _, r := range m.Player1.Tehai.Bara {
        kawa += string(r) + "　"
    }
    for _, g := range m.Player1.Tehai.Groups {
        kawa += g.Word + "　"
    }
    
    s.WriteString(tehaiBorder.Render("Player1 information\n" + kawa))
    
    return s.String()
}
	*/

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) Init() tea.Cmd {
	m.haipai()
	return nil
}