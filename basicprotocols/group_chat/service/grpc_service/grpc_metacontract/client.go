package grpc_metacontract

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "manindexer/basicprotocols/group_chat/service/grpc_service/proto"
	"manindexer/common"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultServerAddr = "localhost:50051"
	defaultTimeout    = 30 * time.Second
)

// Client encapsulates MetaContract gRPC client
type Client struct {
	conn      *grpc.ClientConn
	client    pb.MetaContractServiceClient
	mu        sync.RWMutex
	connected bool
}

// NewClient creates a new gRPC client, connecting to the default address localhost:50051
func NewClient() (*Client, error) {
	return NewClientWithAddress(common.Config.GroupChat.GrpcMetaContractAddress)
}

// NewClientWithAddress creates a new gRPC client, connecting to the specified address
func NewClientWithAddress(addr string) (*Client, error) {
	// Create gRPC connection
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	// Create client
	client := pb.NewMetaContractServiceClient(conn)

	return &Client{
		conn:      conn,
		client:    client,
		connected: true,
	}, nil
}

// Close closes gRPC connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.connected = false
		return c.conn.Close()
	}
	return nil
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// TransferFtRequest is the request parameter for TransferFt method
type TransferFtRequest struct {
	Wif             string
	TokenWif        *string // Optional parameter
	Token           *TokenTransfer
	FtChangeAddress *string  // Optional parameter - FT change address
	OpreturnData    []string // OP_RETURN data
	ChangeAddress   *string  // Optional parameter - BTC change address
}

// Receiver receiver information
type Receiver struct {
	Address string
	Amount  string
}

// TokenTransfer token transfer information
type TokenTransfer struct {
	Receivers  []*Receiver // Multiple receivers
	Codehash   string
	Genesis    string
	GasUtxos   []*GasUtxo
	TokenUtxos []*TokenUtxo
}

// GasUtxo gas UTXO information
type GasUtxo struct {
	TxId        string
	OutputIndex int32
	Satoshis    string
}

// TokenUtxo token UTXO information
type TokenUtxo struct {
	TxId        string
	OutputIndex int32
	TokenAmount string
}

// TransferFtResponse is the response result of TransferFt method
type TransferFtResponse struct {
	Status  string
	Message string
	Data    *TransferFtData
}

// TransferFtData transfer response data
type TransferFtData struct {
	TxId            string
	TxHex           string
	RouteCheckTxHex string
	RouteCheckTxId  string
}

// TransferFt executes token transfer operation
func (c *Client) TransferFt(ctx context.Context, req *TransferFtRequest) (*TransferFtResponse, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is not connected")
	}
	c.mu.RUnlock()

	// Convert request parameters
	pbReq := &pb.TransferFtRequest{
		Wif:             req.Wif,
		TokenWif:        req.TokenWif,
		FtChangeAddress: req.FtChangeAddress,
		OpreturnData:    req.OpreturnData,
		ChangeAddress:   req.ChangeAddress,
	}

	// Convert TokenTransfer
	if req.Token != nil {
		pbReq.Token = &pb.TokenTransfer{
			Codehash: req.Token.Codehash,
			Genesis:  req.Token.Genesis,
		}

		// Convert Receivers
		if len(req.Token.Receivers) > 0 {
			pbReq.Token.Receivers = make([]*pb.Receiver, len(req.Token.Receivers))
			for i, receiver := range req.Token.Receivers {
				pbReq.Token.Receivers[i] = &pb.Receiver{
					Address: receiver.Address,
					Amount:  receiver.Amount,
				}
			}
		}

		// Convert GasUtxos
		if len(req.Token.GasUtxos) > 0 {
			pbReq.Token.GasUtxos = make([]*pb.GasUtxo, len(req.Token.GasUtxos))
			for i, utxo := range req.Token.GasUtxos {
				pbReq.Token.GasUtxos[i] = &pb.GasUtxo{
					TxId:        utxo.TxId,
					OutputIndex: &utxo.OutputIndex,
					Satoshis:    utxo.Satoshis,
				}
			}
		}

		// Convert TokenUtxos
		if len(req.Token.TokenUtxos) > 0 {
			pbReq.Token.TokenUtxos = make([]*pb.TokenUtxo, len(req.Token.TokenUtxos))
			for i, utxo := range req.Token.TokenUtxos {
				pbReq.Token.TokenUtxos[i] = &pb.TokenUtxo{
					TxId:        utxo.TxId,
					OutputIndex: &utxo.OutputIndex,
					TokenAmount: utxo.TokenAmount,
				}
			}
		}
	}

	// Set timeout
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}

	// Call gRPC method
	resp, err := c.client.TransferFt(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call TransferFt: %v", err)
	}

	// Convert response
	result := &TransferFtResponse{
		Status:  resp.Status,
		Message: resp.Message,
	}

	if resp.Data != nil {
		result.Data = &TransferFtData{
			TxId:            resp.Data.TxId,
			TxHex:           resp.Data.TxHex,
			RouteCheckTxHex: resp.Data.RouteCheckTxHex,
			RouteCheckTxId:  resp.Data.RouteCheckTxId,
		}
	}

	return result, nil
}

// TransferFtWithTimeout executes token transfer operation (with timeout)
func (c *Client) TransferFtWithTimeout(req *TransferFtRequest, timeout time.Duration) (*TransferFtResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.TransferFt(ctx, req)
}

// HealthResponse is the response result of GetHealth method
type HealthResponse struct {
	Status  string
	Message string
	Data    *HealthData
}

// HealthData health check response data
type HealthData struct {
	Version     string
	Network     string
	Timestamp   int64
	NoBroadcast bool
}

// GetHealth gets service health status
func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is not connected")
	}
	c.mu.RUnlock()

	// Create request parameters
	pbReq := &pb.HealthRequest{}

	// Set timeout
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}

	// Call gRPC method
	resp, err := c.client.GetHealth(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call GetHealth: %v", err)
	}

	// Convert response
	result := &HealthResponse{
		Status:  resp.Status,
		Message: resp.Message,
	}

	if resp.Data != nil {
		result.Data = &HealthData{
			Version:     resp.Data.Version,
			Network:     resp.Data.Network,
			Timestamp:   resp.Data.Timestamp,
			NoBroadcast: resp.Data.NoBroadcast,
		}
	}

	return result, nil
}

// GetHealthWithTimeout gets service health status (with timeout)
func (c *Client) GetHealthWithTimeout(timeout time.Duration) (*HealthResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.GetHealth(ctx)
}
