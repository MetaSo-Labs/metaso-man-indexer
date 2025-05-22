package metaso

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MetaSo struct {
}

type Tweet struct {
	Id                 string             `json:"id"`
	Number             int64              `json:"number"`
	MetaId             string             `json:"metaid"`
	Address            string             `json:"address"`
	CreateAddress      string             `json:"creator"`
	CreateMetaId       string             `json:"createMetaId"`
	InitialOwner       string             `json:"initialOwner"`
	Output             string             `json:"output"`
	OutputValue        int64              `json:"outputValue"`
	Timestamp          int64              `json:"timestamp"`
	GenesisFee         int64              `json:"genesisFee"`
	GenesisHeight      int64              `json:"genesisHeight"`
	GenesisTransaction string             `json:"genesisTransaction"`
	TxIndex            int                `json:"txIndex"`
	TxInIndex          uint32             `json:"txInIndex"`
	Offset             uint64             `json:"offset"`
	Location           string             `json:"location"`
	Operation          string             `json:"operation"`
	Path               string             `json:"path"`
	ParentPath         string             `json:"parentPath"`
	OriginalPath       string             `json:"originalPath"`
	Encryption         string             `json:"encryption"`
	Version            string             `json:"version"`
	ContentType        string             `json:"contentType"`
	ContentTypeDetect  string             `json:"contentTypeDetect"`
	ContentBody        []byte             `json:"contentBody"`
	ContentLength      uint64             `json:"contentLength"`
	ContentSummary     string             `json:"contentSummary"`
	Status             int                `json:"status"`
	OriginalId         string             `json:"originalId"`
	IsTransfered       bool               `json:"isTransfered"`
	Preview            string             `json:"preview"`
	Content            string             `json:"content"`
	Pop                string             `json:"pop"`
	PopLv              int                `json:"popLv"`
	ChainName          string             `json:"chainName"`
	DataValue          int                `json:"dataValue"`
	Mrc20MintId        []string           `json:"mrc20MintId"`
	MogoID             primitive.ObjectID `bson:"_id,omitempty"`
	LikeCount          int                `json:"likeCount" bson:"likecount"`
	CommentCount       int                `json:"commentCount" bson:"commentcount"`
	ShareCount         int                `json:"shareCount" bson:"sharecount"`
	Hot                int                `json:"hot" bson:"hot"`
	DonateCount        int                `json:"donateCount" bson:"donatecount"`
	Host               string             `json:"host"`
	Keywords           []string           `json:"keywords"`
	Blocked            bool               `json:"blocked"`
	IsRecommended      bool               `json:"is_recommended"`
}
type SyncLastId struct {
	Tweet        primitive.ObjectID `bson:"tweet"`
	TweetLike    primitive.ObjectID `bson:"tweetlike"`
	TweetComment primitive.ObjectID `bson:"tweetcomment"`
}
type TweetLike struct {
	PinId         string `json:"pinId" bson:"pinid"`
	PinNumber     int64  `json:"pinNumber" bson:"pinnumber"`
	ChainName     string `json:"chainName" bson:"chainname"`
	LikeToPinId   string `json:"likeToPinId" bson:"liketopinid"`
	CreateAddress string `json:"createAddress" bson:"createaddress"`
	CreateMetaid  string `json:"CreateMetaid" bson:"createmetaid"`
	IsLike        string `json:"isLike" bson:"islike"`
	Timestamp     int64  `json:"timestamp" bson:"timestamp"`
}
type TweetComment struct {
	PinId         string `json:"pinId" bson:"pinid"`
	PinNumber     int64  `json:"pinNumber" bson:"pinnumber"`
	ChainName     string `json:"chainName" bson:"chainname"`
	CommentPinId  string `json:"commentToPinId" bson:"commentpinid"`
	CreateAddress string `json:"createAddress" bson:"createaddress"`
	CreateMetaid  string `json:"CreateMetaid" bson:"createmetaid"`
	Content       string `json:"content" bson:"content"`
	ContentType   string `json:"contentType" bson:"contenttype"`
	Timestamp     int64  `json:"timestamp" bson:"timestamp"`
}
type PinLike struct {
	IsLike string `json:"isLike" bson:"islike"`
	LikeTo string `json:"likeTo" bson:"liketo"`
}
type PinComment struct {
	CommentTo   string `json:"commentTo" bson:"commentto"`
	Content     string `json:"content" bson:"content"`
	ContentType string `json:"contentType" bson:"contenttype"`
}
type HostData struct {
	Host        string `json:"host" bson:"host"`
	BlockHeight int64  `json:"blockHeight" bson:"blockHeight"`
	BlockHash   string `json:"blockHash" bson:"blockHash"`
	TxCount     int64  `json:"txCount" bson:"txCount"`
	TxSize      int64  `json:"txSize" bson:"txSize"`
	TxFee       int64  `json:"txFee" bson:"txFee"`
}

type PayBuzz struct {
	PublicContent  string   `json:"publicContent"`
	EncryptContent string   `json:"encryptContent"`
	ContentType    string   `json:"contentType"`
	PublicFiles    []string `json:"publicFiles"`
	EncryptFiles   []string `json:"encryptFiles"`
}
type Mrc20DeployInfo struct {
	MogoID       primitive.ObjectID  `bson:"_id,omitempty"`
	Tick         string              `json:"tick"`
	TokenName    string              `json:"tokenName"`
	Decimals     string              `json:"decimals"`
	AmtPerMint   string              `json:"amtPerMint"`
	MintCount    uint64              `json:"mintCount"`
	BeginHeight  string              `json:"beginHeight"`
	EndHeight    string              `json:"endHeight"`
	Metadata     string              `json:"metadata"`
	DeployType   string              `json:"type"`
	PremineCount uint64              `json:"premineCount"`
	PinCheck     Mrc20DeployQual     `json:"pinCheck"`
	PayCheck     Mrc20DeployPayCheck `json:"payCheck"`
	TotalMinted  uint64              `json:"totalMinted"`
	Mrc20Id      string              `json:"mrc20Id"`
	PinNumber    int64               `json:"pinNumber"`
	Chain        string              `json:"chain"`
	Holders      uint64              `json:"holders"`
	TxCount      uint64              `json:"txCount"`
	MetaId       string              `json:"metaId"`
	Address      string              `json:"address"`
	DeployTime   int64               `json:"deployTime"`
	IdCoin       int                 `json:"idCoin"`
}
type Mrc20DeployQual struct {
	Creator string `json:"creator"`
	Lv      string `json:"lvl"`
	Path    string `json:"path"`
	Count   string `json:"count"`
}
type Mrc20DeployPayCheck struct {
	PayTo     string `json:"payTo"`
	PayAmount string `json:"payAmount"`
}
type MempoolData struct {
	Path          string `json:"path"`
	PinId         string `json:"pinId"`
	CreateTime    int64  `json:"createTime"`
	Target        string `json:"target"`
	Content       string `json:"content"`
	IsCancel      int    `json:"isCancel"`
	CreateMetaId  string `json:"createMetaId"`
	CreateAddress string `json:"createAddress"`
}

// PIN Engagement Value
type PEVData struct {
	Host             string          `json:"host"`
	FromPINId        string          `json:"fromPINId"`
	ToPINId          string          `json:"toPINId"`
	Path             string          `json:"path"`
	Address          string          `json:"address"`
	MetaId           string          `json:"metaId"`
	FromChainName    string          `json:"fromChainName"`
	ToChainName      string          `json:"toChainName"`
	MetaBlockHeight  int64           `json:"metaBlockHeight"`
	StartBlockHeight int64           `json:"startBlockHeight"`
	EndBlockHeight   int64           `json:"endBlockHeight"`
	BlockHeight      int64           `json:"blockHeight"`
	IncrementalValue decimal.Decimal `json:"incrementalValue"`
	Poplv            int             `json:"poplv"`
}
type LastMetaBlockData struct {
	BlockData  MetaBlockData `json:"blockData"`
	LastNumber int64         `json:"lastNumber"`
	Step       int           `json:"step"`
	Init       int64         `json:"init"`
}
type MetaBlockData struct {
	Header          string               `json:"header"`
	PreHeader       string               `json:"preHeader"`
	MetablockHeight int64                `json:"metablockHeight"`
	Chains          []MetaBlockChainData `json:"chains"`
	OnChain         string               `json:"onChain"`
	Timestamp       int64                `json:"timestamp"`
	TxHash          string               `json:"txHash"`
	TxIndex         int                  `json:"txIndex"`
}
type MetaBlockChainData struct {
	Chain string `json:"chain"`
	//PreEndBlock string `json:"preEndBlock"`
	PreEndBlock string `json:"lastBlock"`
	StartBlock  string `json:"startBlock"`
	EndBlock    string `json:"endBlock"`
}

// MetaSoMDV
type MetaSoMDV struct {
	MetaId    string          `json:"metaId"`
	Address   string          `json:"address"`
	DataValue decimal.Decimal `json:"dataValue"`
}

// MteaSoNDV
type MetaSoNDV struct {
	Host      string          `json:"host"`
	DataValue decimal.Decimal `json:"dataValue"`
}

// MetaSoBlockInfo
type MetaSoBlockInfo struct {
	Block            int64           `json:"block"`
	HistoryValue     decimal.Decimal `json:"historyValue"`
	DataValue        decimal.Decimal `json:"dataValue"`
	PinNumber        int64           `json:"pinNumber"`
	PinNumberHasHost int64           `json:"pinNumberHasHost"`
	AddressNumber    int64           `json:"addressNumber"`
	HostNumber       int64           `json:"hostNumber"`
	MetaBlock        MetaBlockData   `json:"metaBlock"`
	BlockTime        int64           `json:"blockTime"`
}

// MetaSoBlockMDV
type MetaSoBlockMDV struct {
	MetaId           string          `json:"metaId"`
	Address          string          `json:"address"`
	Block            int64           `json:"block"`
	HistoryValue     decimal.Decimal `json:"historyValue"`
	DataValue        decimal.Decimal `json:"dataValue"`
	PinNumber        int64           `json:"pinNumber"`
	PinNumberHasHost int64           `json:"pinNumberHasHost"`
	BlockTime        int64           `json:"blockTime"`
}

// MetaSoBlockNDV
type MetaSoBlockNDV struct {
	Host         string          `json:"host"`
	Block        int64           `json:"block"`
	HistoryValue decimal.Decimal `json:"historyValue"`
	DataValue    decimal.Decimal `json:"dataValue"`
	PinNumber    int64           `json:"pinNumber"`
	BlockTime    int64           `json:"blockTime"`
}
type MetaSoHostAddress struct {
	Host             string          `json:"host"`
	MetaId           string          `json:"metaId"`
	Address          string          `json:"address"`
	Block            int64           `json:"block"`
	HistoryValue     decimal.Decimal `json:"historyValue"`
	DataValue        decimal.Decimal `json:"dataValue"`
	PinNumber        int64           `json:"pinNumber"`
	PinNumberHasHost int64           `json:"pinNumberHasHost"`
	BlockTime        int64           `json:"blockTime"`
}

func OctalStringToDecimal(octalStr string, intNum int, divisor float64) (*float64, error) {
	decimalNum := new(big.Int)
	base := big.NewInt(8)
	for _, char := range octalStr {
		digit := int64(char - '0')
		if digit < 0 || digit > 7 {
			return nil, fmt.Errorf("err: %c", char)
		}

		decimalNum.Mul(decimalNum, base)
		decimalNum.Add(decimalNum, big.NewInt(digit))
	}
	bigIntStrFull := decimalNum.String()
	bingIntStr := ""
	if len(bigIntStrFull) > intNum {
		bingIntStr = bigIntStrFull[:intNum]
	} else {
		bingIntStr = bigIntStrFull
	}
	firstFourInt, err := strconv.Atoi(bingIntStr)
	if err != nil {
		return nil, err
	}
	result := float64(firstFourInt) / divisor
	//rounded := math.Round(result*10000) / 10000
	return &result, nil
}

type MetasoDonate struct {
	PinId         string          `json:"pinId" bson:"pinid"`
	PinNumber     int64           `json:"pinNumber" bson:"pinnumber"`
	ChainName     string          `json:"chainName" bson:"chainname"`
	CreateAddress string          `json:"createAddress" bson:"createaddress"`
	CreateMetaid  string          `json:"CreateMetaid" bson:"createmetaid"`
	Timestamp     int64           `json:"timestamp" bson:"timestamp"`
	CreateTime    string          `json:"createTime" bson:"createtime"`
	ToAddress     string          `json:"toAddress" bson:"toaddress"`
	CoinType      string          `json:"coinType" bson:"cointype"`
	Amount        decimal.Decimal `json:"amount" bson:"amount"`
	ToPin         string          `json:"toPin" bson:"topin"`
	Message       string          `json:"message" bson:"message"`
}

type BlockedSetting struct {
	BlockedType     string `json:"blockedType"`
	BlockedContent  string `json:"blockedContent"`
	Timestamp       int64  `json:"timestamp"`
	OriginalContent string `json:"originalContent"`
}

// RecommendedAuthor represents an author who is recommended
type RecommendedAuthor struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AuthorID   string             `bson:"author_id" json:"authorId"` // ID of the recommended author
	AuthorName string             `bson:"author_name" json:"authorName"`
	CreatedAt  time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updatedAt"`
}
type PostBase struct {
	MongoId       primitive.ObjectID `bson:"_id,omitempty" json:"mongoId"`
	Id            string             `json:"id"`
	Number        int64              `json:"number"`
	MetaId        string             `json:"metaid"`
	Address       string             `json:"address"`
	CreateAddress string             `json:"creator"`
	CreateMetaId  string             `json:"createMetaId"`
	Blocked       bool               `json:"blocked"`
	IsRecommended bool               `json:"is_recommended"`
}
