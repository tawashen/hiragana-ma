package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort" // 修正: CreateGroup関数でsort.Sortを使用しているため追加
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
	Pais []string //構成文字（サジェスト用）
	//RegisteredBy string //登録したプレイヤー名
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
	Yama []string // 修正: []rune → []string（山札を文字列スライスで管理）
	WordDic map[string]*Word //有効なワード辞書（プレイヤーが登録）
	YakuDic map[string]*Yaku //役名→役情報
	WordYakuDic []WordYaku //単語と役の紐付け履歴（学習データ）
	CurrentPlayer int //1 or 2
	Turn int
	Phase string //ツモ、捨て牌、グループ作成、上がり判定など
    TumoHai string
	Alphabet []string
	InputBuffer string
	ChoiceIndex []int
}

type Player struct {
	Name string
	Tehai *Tehai
	Kawa []string // 修正: []rune → []string（捨て牌を文字列スライスで管理）
	Score int
	RichiFlag bool //リーチ的な何か用
}

type Tehai struct {
	Bara []string // 修正: []rune → []string（未グループ化の牌を文字列スライスで管理）
	Groups []Group //グループ化された単語
}

type Group struct {
	Word string //グループ化された文字列
	Pais []string //構成文字
	Comp bool //単語として登録するか？
	//NakiFrom string //誰から鳴いたか（プレイヤー名）
}

// 山札を初期化する関数
func createYama() []string {
	// 修正: []rune → []string（50音を文字列スライスで管理）
	gojuon := []string{
		"あ", "い", "う", "え", "お",
		"か", "き", "く", "け", "こ",
		"さ", "し", "す", "せ", "そ",
		"た", "ち", "つ", "て", "と",
		"な", "に", "ぬ", "ね", "の",
		"は", "ひ", "ふ", "へ", "ほ",
		"ま", "み", "む", "め", "も",
		"や", "ゆ", "よ",
		"ら", "り", "る", "れ", "ろ",
		"わ", "を", "ん",
	}

	yama := []string{}

	// 50音を4セット追加
	for i := 0; i < 4; i++ {
		yama = append(yama, gojuon...)
	}

	// 濁点（゛）を10枚
	for i := 0; i < 10; i++ {
		yama = append(yama, "゛")
	}

	// 半濁点（゜）を10枚
	for i := 0; i < 10; i++ {
		yama = append(yama, "゜")
	}

	// 伸ばし棒（ー）を10枚
	for i := 0; i < 10; i++ {
		yama = append(yama, "ー")
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
		fmt.Printf("error %v", err) // 修正: &v → %v（フォーマット指定子の誤り）
	}
}

func initialModel() Model {
	player1 := Player {
		Name: "Player1",
        Tehai: &Tehai{
            Bara: []string{}, // 修正: []rune{} → []string{}
            Groups: []Group{},
        },
		Kawa: []string{}, // 修正: []rune{} → []string{}
		Score: 0,
		RichiFlag: false,
	}

		player2 := Player {
		Name: "Player2",
        Tehai: &Tehai{
            Bara: []string{}, // 修正: []rune{} → []string{}
            Groups: []Group{},
        },
		Kawa: []string{}, // 修正: []rune{} → []string{}
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
		TumoHai: "",
		Alphabet: []string{},  // 誤: "" → 正: []string{}（空のスライス）
		InputBuffer: "",
		ChoiceIndex: []int{},
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


func (m Model) Init() tea.Cmd {
	m.haipai()
	return nil
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
	menu2 := ""
	switch m.Phase {
	case "ツモ前":
		menu = "１：ツモ"
		//チーはツモ牌表示前のみ可能
		//だけど無限に長くするということなら一旦はグループ扱いにしなければならないわけで、そうなると単語確定とグループ化を分ける必要があると今更気づいた
		//つまり単語登録は上がった時にのみまとめて登録されるようにしないといかんか
		//チー？の後はツモ出来ずに次のプレイヤーへ？
	case "ツモ中":
		menu = "１：スルー　アルファベット：捨て牌　２：バラし"
	case "グループ化":
		menu = "１：終了　アルファベット：選択 ２：バラし"
		for _, index := range m.ChoiceIndex {//選択中の牌を表示
			menu2 += m.Player1.Tehai.Bara[index]
		}
	case "単語化？":
		menu = "Ｙ：単語登録　その他：グループ化のみ" 
	}
	s.WriteString(textStyle.Render(fmt.Sprintf("%s\n", menu)))
	s.WriteString(textStyle.Render(fmt.Sprintf("%s\n", menu2)))
    s.WriteString("\n\n\n")
    

    kawaBuilder := strings.Builder{}
    for _, pai := range m.Player1.Tehai.Bara {
        kawaBuilder.WriteString(paiStyle.Render(pai)) // 修正: string(r) → pai（既にstring型）
        kawaBuilder.WriteString(" ") // 牌の間にスペース
    }
    
    for _, g := range m.Player1.Tehai.Groups {
        kawaBuilder.WriteString(wordStyle.Render(g.Word))
        kawaBuilder.WriteString(" ")
    }

	// 修正: 選択用アルファベットを1つずつ表示
	alphabetBuilder := strings.Builder{}
	for index, alpha := range m.Alphabet {
		alphabetBuilder.WriteString(paiStyle.Render(alpha))
		alphabetBuilder.WriteString(" ")
		if index == len(m.Player1.Tehai.Bara) -1 {
			break
		}
	}
	s.WriteString(alphabetBuilder.String())
	s.WriteString("\n")
	
    
    s.WriteString(kawaBuilder.String())
    
    return s.String()

}




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
//            case "2"://単語作りへ
//                return m.MakeWord(), nil
            }
        }
        
        if m.Phase == "ツモ中" {
            // 修正: m.Alphabetを使って判定（[]stringになった）
            key := msg.String()
			if key == "1" {//スルー
				m.Phase = "グループ化"
				m.TumoHai = ""
				return m, nil
			}
            for i, alpha := range m.Alphabet {
                if key == alpha {
                    return m.DiscardPai(i), nil
                }
            }

			if key == "2" {
				//m.Phase = "バラし"
				return m.BaraBara(), nil
			}
        }


		if m.Phase == "単語化？" {
			switch msg.String() {
			case "y", "Y":
				m = m.CreateGroup(m.InputBuffer, true)
				m.InputBuffer = ""
				m.ChoiceIndex = []int{}
				m.Phase = "グループ化"
				return m, nil

			default:
				m = m.CreateGroup(m.InputBuffer, false)
				m.InputBuffer = ""
				m.Phase = "グループ化"
				return m, nil
			}
		}

		//if m.Phase == "バラし" {

		//}

		if m.Phase == "グループ化" {
			switch msg.String() {
			case "1":
				m.Phase = "ツモ前"
				m.CurrentPlayer = 1
				return m, nil

			case "2": 
				//m.Phase = "バラし"
				return m.BaraBara(), nil
			

			case "enter":
				//if len(m.InputBuffer) == 3 {
				//m = m.CreateGroup(m.InputBuffer)
				//m.InputBuffer = ""
				m.Phase = "単語化？"
				//}
				
				return m, nil


			case "backspace":
				if len(m.InputBuffer) > 0 {
					m.InputBuffer = m.InputBuffer[:len(m.InputBuffer)-1]
					m.ChoiceIndex = m.ChoiceIndex[:len(m.ChoiceIndex) -1]
				}
				return m, nil
			
			default:
			    key := msg.String()
    // 1文字の半角アルファベットかチェック
    			if len(key) == 1 {
        			char := rune(key[0])
        // 'a'から始まるインデックスを計算
        			index := int(char - 'a')
        // 範囲内かチェック（0以上、かつBaraの長さ未満）
        			if index >= len(m.Player1.Tehai.Bara) {
            return m, nil
        		}
			}

				for index, alpha := range m.Alphabet {
					// 修正: alphaは既にstring型
					if key == alpha {
						m.InputBuffer += key
						m.ChoiceIndex = append(m.ChoiceIndex, index)
						break
					}
				}
				return m, nil
			
			}
    	}
	}
    return m, nil
}


























func (m *Model) DiscardPai(index int) Model {
	m.Player1.Tehai.Bara = append(m.Player1.Tehai.Bara[:index], m.Player1.Tehai.Bara[index+1:]...)
	// 修正: TumoHaiは既にstring型なのでそのまま追加
	if m.TumoHai != "" {
		m.Player1.Tehai.Bara = append(m.Player1.Tehai.Bara, m.TumoHai)
	}
	m.TumoHai = ""
	m.Phase = "グループ化"
	return *m
}

func (m *Model) Tumo() Model {
	m.Phase = "ツモ中"
    m.TumoHai = m.Yama[0]
	m.Yama = m.Yama[1:]

	m.updateAlphabet()

	return *m
}


func (m *Model) MakeWord() Model {
    m.Phase = "グループ化"  // のための文字列スライスをModelに埋める
    
    // 修正: 半角アルファベットを使用（キーボード入力に対応）
    alphabet := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"}
    
    ab := []string{}
    for i := range m.Player1.Tehai.Bara {
        ab = append(ab, alphabet[i])
    }
    
    m.Alphabet = ab  // 選択表示用アルファベット文字列スライス
    return *m
}




// バッファの内容から面子を作る
func (m *Model) CreateGroup(buffer string, comp bool) Model {

	//m.ChoiceIndex = []int{}
    group := Group{
        Word: "",
        Pais: []string{},
		Comp: comp,
    }
    
    // 修正: 半角アルファベット（a-n）をインデックスに変換
    indices := []int{}
    for _, char := range buffer {
        // 半角アルファベットの場合
        if char >= 'a' && char <= 'n' {
            index := int(char - 'a')
            if index >= 0 && index < len(m.Player1.Tehai.Bara) {
                indices = append(indices, index)
            }
        }
    }
    
	//m.ChoiceIndex = indices //選択中牌の表示用Index

    for _, index := range indices {
        pai := m.Player1.Tehai.Bara[index]
        group.Pais = append(group.Pais, pai)
        group.Word += pai
	}
    // インデックスを降順にソート（後ろから削除するため）
    sort.Sort(sort.Reverse(sort.IntSlice(indices)))
    
    // 選択された牌をGroupに追加して、Baraから削除
    for _, index := range indices {
        // Baraから削除
        m.Player1.Tehai.Bara = append(
            m.Player1.Tehai.Bara[:index],
            m.Player1.Tehai.Bara[index+1:]...,
        )
    }
    
    // Groupsに追加
    m.Player1.Tehai.Groups = append(m.Player1.Tehai.Groups, group)

	// Trueなら単語として辞書登録
	if group.Comp == true {
		if _, exist := m.WordDic[group.Word]; !exist {
			m.WordDic[group.Word] = &Word{
				Text: group.Word,
				Pais: group.Pais,
			}
		}
	}
    
    return *m
}


func (m *Model) BaraBara() Model {
	for _, group := range m.Player1.Tehai.Groups {
		for _, pai := range group.Pais {
			m.Player1.Tehai.Bara = append(m.Player1.Tehai.Bara, pai)
		}
	}
	m.Player1.Tehai.Groups = []Group{}

	m.updateAlphabet()
	
	return *m
}

// Alphabetを更新するヘルパーメソッド
func (m *Model) updateAlphabet() {
    alphabet := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"}
    ab := []string{}
    for i := range m.Player1.Tehai.Bara {
        if i < len(alphabet) {
            ab = append(ab, alphabet[i])
        }
    }
    m.Alphabet = ab
}
