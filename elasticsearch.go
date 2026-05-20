package dbkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/elastic/go-elasticsearch/v9"
)

// ESClient 对官方 elasticsearch 客户端的薄封装。
type ESClient struct {
	es *elasticsearch.Client
}

func openElasticsearch(ctx context.Context, cfg ElasticsearchConfig, log Logger) (*ESClient, error) {
	cfg = cfg.withDefaults()
	if len(cfg.Addresses) == 0 && cfg.CloudID == "" {
		return nil, fmt.Errorf("dbkit elasticsearch: addresses or cloud_id is required when enabled")
	}

	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
		Transport: &http.Transport{
			ResponseHeaderTimeout: cfg.Dial,
		},
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("dbkit elasticsearch: new client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.Dial)
	defer cancel()
	res, err := es.Ping(es.Ping.WithContext(pingCtx))
	if err != nil {
		return nil, fmt.Errorf("dbkit elasticsearch: ping: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("dbkit elasticsearch: ping status %s: %s", res.Status(), string(body))
	}

	log.Info("elasticsearch connected", String("component", "elasticsearch"))
	return &ESClient{es: es}, nil
}

func (c *ESClient) Raw() *elasticsearch.Client { return c.es }

func (c *ESClient) Ping(ctx context.Context) error {
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch ping: %s", string(body))
	}
	return nil
}

func (c *ESClient) Close() error {
	// 官方客户端无显式 Close，Transport 由 GC 回收；此处为生命周期对称占位。
	return nil
}

// IndexDocument 索引单条文档（若不存在则创建，存在则覆盖，取决于 index 设置）。
func (c *ESClient) IndexDocument(ctx context.Context, index, docID string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("dbkit elasticsearch: marshal: %w", err)
	}
	res, err := c.es.Index(
		index,
		bytes.NewReader(data),
		c.es.Index.WithContext(ctx),
		c.es.Index.WithDocumentID(docID),
	)
	if err != nil {
		return fmt.Errorf("dbkit elasticsearch: index: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("dbkit elasticsearch: index status %s: %s", res.Status(), string(b))
	}
	return nil
}

// GetDocument 按 ID 获取文档。
func (c *ESClient) GetDocument(ctx context.Context, index, docID string) ([]byte, error) {
	res, err := c.es.Get(index, docID, c.es.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("dbkit elasticsearch: get: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("dbkit elasticsearch: get status %s: %s", res.Status(), string(b))
	}
	return io.ReadAll(res.Body)
}

// DeleteDocument 按 ID 删除文档。
func (c *ESClient) DeleteDocument(ctx context.Context, index, docID string) error {
	res, err := c.es.Delete(index, docID, c.es.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("dbkit elasticsearch: delete: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("dbkit elasticsearch: delete status %s: %s", res.Status(), string(b))
	}
	return nil
}

// CreateIndex 创建索引（body 为 mapping/settings JSON 对象）。
func (c *ESClient) CreateIndex(ctx context.Context, index string, body map[string]any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	res, err := c.es.Indices.Create(
		index,
		c.es.Indices.Create.WithContext(ctx),
		c.es.Indices.Create.WithBody(&buf),
	)
	if err != nil {
		return fmt.Errorf("dbkit elasticsearch: create index: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("dbkit elasticsearch: create index status %s: %s", res.Status(), string(b))
	}
	return nil
}
