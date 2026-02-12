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
    TumoHai string
	Alphabet string
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
		Phase: "ツモ前",
		TumoHai: ""
		Alphabet: ""
		}

		return m

	}


func (m *Model) haipai() Model {
	for i := 0; i < 13; i++ {
		m.Player1.Tehai.Bara = append(m.Player1.Tehai.Bara, m.Yama[i])
		//m.Yamaからindexの要素を削除する。本当は直接YamaからTehai.Baraに移動できたらてっとり早いが
	}
	for i := 0; i < 13; i++ {
		m.Player2.Tehai.Bara = append(m.Player2.Tehai.Bara, m.Yama[i])
	    //Playersにしてなかったのでイテレート出来ないｗ
	}


	m.Yama = m.Yama[26:]
	return *m
}



func (m Model) View() string {
    var s strings.Builder
    
    s.WriteString(playerStyle.Render("Player2 information\n")) 
    s.WriteString(textStyle.Render("Player2 river\n"))
    s.WriteString("\n\n\n")
    
    s.WriteString(textStyle.Render(fmt.Sprintf("　　　　Turn:%d\n", m.Turn)))
    s.WriteString(textStyle.Render(fmt.Sprintf("　　　　Phase:%s\n", m.Phase)))
	s.WriteString(textStyle.Render(fmt.Sprintf("　　　　ツモ牌:%s\n", m.TumoHai)))
    s.WriteString("\n")
    
	menu := ""
	switch m.Phase {
	case "ツモ前":
		menu = "１：ツモ　２：単語作り　３：チー？"
		//チーはツモ牌表示前のみ可能
		//だけど無限に長くするということなら一旦はグループ扱いにしなければならないわけで、そうなると単語確定とグループ化を分ける必要があると今更気づいた
		//つまり単語登録は上がった時にのみまとめて登録されるようにしないといかんか
		//チー？の後はツモ出来ずに次のプレイヤーへ？
	case "ツモ中":
		menu = "１：スルー　アルファベット：捨て牌"
	case "グループ作り":
		menu = "１：終了　アルファベット：選択"
		//1文字じゃなくて3文字まで一度で選択とか可能なのか？
	}
	s.WriteString(textStyle.Render(fmt.Sprintf))
    
    s.WriteString("\n\n\n")
    

    kawaBuilder := strings.Builder{}
    for _, r := range m.Player1.Tehai.Bara {
        kawaBuilder.WriteString(paiStyle.Render(string(r)))
        kawaBuilder.WriteString(" ") // 牌の間にスペース
    }
    
    for _, g := range m.Player1.Tehai.Groups {
        kawaBuilder.WriteString(wordStyle.Render(g.Word))
        kawaBuilder.WriteString(" ")
    }

	//選択用アルファベットはいつでも表示にしておく
	s.WriteString(paiStyle.Render(m.Alphabet))
	
    
    s.WriteString(kawaBuilder.String())
    
    return s.String()

}


var (
	textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#000000"))
	playerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Background(lipgloss.Color("#000000"))
	//waterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	//mtStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B4513"))
)

	// 基本的な枠付きスタイルの作り方
var boxStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()). // 角丸の枠
    BorderForeground(lipgloss.Color("255")). // 枠の色（白）
    Padding(1, 2) // 内側の余白（上下1、左右2）

var paiStyleWithBorder = lipgloss.NewStyle().
    Border(lipgloss.NormalBorder()).       // 枠を付ける
    BorderForeground(lipgloss.Color("240")). // グレーの枠
    Background(lipgloss.Color("255")).     // 白い背景
    Foreground(lipgloss.Color("0")).       // 黒い文字
    Width(2).
    Align(lipgloss.Center).
    Bold(true)

	var paiStyle = lipgloss.NewStyle().
    Background(lipgloss.Color("255")).     // 白い背景
    Foreground(lipgloss.Color("0")).       // 黒い文字
    Width(2).                              // 全角1文字分の幅
    Align(lipgloss.Center).                // 中央揃え
	//Padding(0, 0).
	//PaddingRight(1).
    Bold(true)                             // 太字で見やすく

	// 単語用：同じく白背景に黒文字
var wordStyle = lipgloss.NewStyle().
    Background(lipgloss.Color("255")).     // 白い背景
    Foreground(lipgloss.Color("0")).       // 黒い文字
    Padding(0, 1).                         // 左右に余白
    Bold(true)



// さらにシンプルに（全角アルファベットの場合）：
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q", "esc":
            return m, tea.Quit
        }
        
        if m.Phase == "ツモ前" {
            switch msg.String() {
            case "1"://ツモへ
                return m.Tumo(), nil
            case "2"://単語作りへ
                return m.MakeWord(), nil
            }
        }
        
        if m.Phase == "ツモ中" {
            // m.Alphabetを使って判定（MakeWordで生成したもの）
            key := msg.String()
            for i, alpha := range m.Alphabet {
                if key == alpha {
                    return m.DiscardPai(i), nil
                }
            }
        }
    }
    return m, nil
}


func (m *Model) DiscardPai(index) Model {
	m.Player1.Tehai = append(m.Player1.Tehai[:index], m.Player1.Tehai[index+1:]...)
	m.Playser1.Tehai = append(m.Player1.Tehai, m.TumoHai)
	m.TumoHai = ""
	m.Phase = "グループ作り"
}

func (m *Model) Tumo() Model {
	m.Phase = "ツモ中"
    m.TumoHai = string(m.Yama[0])
	m.Yama = m.Yama[1:]

    alphabet := []rune{'Ａ', 'Ｂ', 'Ｃ', 'Ｄ', 'Ｅ', 'Ｆ', 'Ｇ', 'Ｈ', 'Ｉ', 'Ｊ', 'Ｋ', 'Ｌ', 'Ｍ', 'Ｎ'}
    
    ab := []string{}
    for i := range m.Player1.Tehai.Bara {
        ab = append(ab, string(alphabet[i]))
    }
    
    m.Alphabet = ab//選択表示用アルファベット文字列
	return *m
}


func (m *Model) MakeWord() Model {
    m.Phase = "グループ作り"//のための文字列スライスをModelに埋める
    
    alphabet := []rune{'Ａ', 'Ｂ', 'Ｃ', 'Ｄ', 'Ｅ', 'Ｆ', 'Ｇ', 'Ｈ', 'Ｉ', 'Ｊ', 'Ｋ', 'Ｌ', 'Ｍ', 'Ｎ'}
    
    ab := []string{}
    for i := range m.Player1.Tehai.Bara {
        ab = append(ab, string(alphabet[i]))
    }
    
    m.Alphabet = ab//選択表示用アルファベット文字列
    return *m
}


func (m Model) Init() tea.Cmd {
	m.haipai()
	return nil
}


ツモ前
　１：ツモ　２：単語作り
　　ツモ　メソッド　→ツモ後
　　単語作り　メソッド　→ツモ前

ツモ後
　１：捨て牌　２：単語作り　３：上がり
　捨て牌
　　１：捨て牌選び　２：単語作り
　　　捨て牌選び　メソッド　→単語作り
　単語作り　メソッド　
　　１：A~H？（グループ数による）　２：終了
　上がり　メソッド

type Model struct {
    // ... 他のフィールド ...
    InputBuffer string  // キー入力を溜めるバッファ
    Phase string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        key := msg.String()
        
        if m.Phase == "グループ化" {
            switch key {
            case "ctrl+c", "q", "esc":
                return m, tea.Quit
            
            case "enter":
                // Enterで確定：バッファの内容を処理
                if len(m.InputBuffer) > 0 {
                    m = m.CreateGroup(m.InputBuffer)
                    m.InputBuffer = ""  // バッファをクリア
                }
                return m, nil
            
            case "backspace":
                // バックスペースで1文字削除
                if len(m.InputBuffer) > 0 {
                    m.InputBuffer = m.InputBuffer[:len(m.InputBuffer)-1]
                }
                return m, nil
            
            default:
                // a~nなどの有効なキーならバッファに追加
                for _, alpha := range m.Alphabet {
                    if key == alpha {
                        m.InputBuffer += key
                        break
                    }
                }
                return m, nil
            }
        }
    }
    return m, nil
}

// バッファの内容から面子を作る
func (m *Model) CreateGroup(buffer string) Model {
    group := Group{
        Word: "",
        Pais: []string{},
    }
    
    // バッファの各文字（a, b, c...）をインデックスに変換
    indices := []int{}
    for _, char := range buffer {
        // 'a' = 0, 'b' = 1, 'c' = 2...
        index := int(char - 'a')
        if index >= 0 && index < len(m.Player1.Tehai.Bara) {
            indices = append(indices, index)
        }
    }
    
    // インデックスを降順にソート（後ろから削除するため）
    sort.Sort(sort.Reverse(sort.IntSlice(indices)))
    
    // 選択された牌をGroupに追加して、Baraから削除
    for _, index := range indices {
        pai := m.Player1.Tehai.Bara[index]
        group.Pais = append(group.Pais, pai)
        group.Word += pai
        
        // Baraから削除
        m.Player1.Tehai.Bara = append(
            m.Player1.Tehai.Bara[:index],
            m.Player1.Tehai.Bara[index+1:]...,
        )
    }
    
    // Groupsに追加
    m.Player1.Tehai.Groups = append(m.Player1.Tehai.Groups, group)
    
    return *m
}

// View()で入力バッファを表示
func (m Model) View() string {
    var s strings.Builder
    
    // ... 他の表示 ...
    
    if m.Phase == "グループ化" {
        s.WriteString("\n選択中: ")
        s.WriteString(m.InputBuffer)  // 入力中の文字を表示（例: "abc"）
        s.WriteString("\n(Enter: 確定, Backspace: 削除, Esc: キャンセル)")
    }
    
    return s.String()
}

// より視覚的に表示する例
func (m Model) View() string {
    var s strings.Builder
    
    if m.Phase == "グループ化" {
        // 牌を表示（選択中のものをハイライト）
        for i, pai := range m.Player1.Tehai.Bara {
            alpha := string(rune('a' + i))
            
            // InputBufferに含まれているかチェック
            isSelected := false
            for _, char := range m.InputBuffer {
                if string(char) == alpha {
                    isSelected = true
                    break
                }
            }
            
            if isSelected {
                // 選択中は色を変える
                selectedStyle := lipgloss.NewStyle().
                    Background(lipgloss.Color("86")).  // 水色背景
                    Foreground(lipgloss.Color("0")).
                    Bold(true)
                s.WriteString(selectedStyle.Render(pai))
            } else {
                s.WriteString(paiStyle.Render(pai))
            }
            s.WriteString(" ")
        }
        
        s.WriteString("\n\n選択: " + m.InputBuffer)
        s.WriteString("\n(Enter: 確定)")
    }
    
    return s.String()
}
