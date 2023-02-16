package chatgpt

import (
	"context"
	"time"

	gogpt "github.com/sashabaranov/go-gpt3"
)

type ChatGPT struct {
	client         *gogpt.Client
	ctx            context.Context
	userId         string
	maxQuestionLen int
	maxText        int
	maxAnswerLen   int
	timeOut        time.Duration // 超时时间, 0表示不超时
	doneChan       chan struct{}
	cancel         func()

	ChatContext *ChatContext
}

func New(ApiKey, UserId string, timeOut time.Duration) *ChatGPT {
	var ctx context.Context
	var cancel func()
	if timeOut == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), timeOut)
	}
	timeOutChan := make(chan struct{}, 1)
	go func() {
		<-ctx.Done()
		timeOutChan <- struct{}{} // 发送超时信号，或是提示结束，用于聊天机器人场景，配合GetTimeOutChan() 使用
	}()

	//  接口限制
	//error, status code: 400, message: This model's maximum context length is 4097 tokens, however you requested 4218 tokens (122 in your prompt; 4096 for the completion). Please reduce your prompt; or com
	const maxToken = 4096
	maxAnswer := 1024

	return &ChatGPT{
		client:         gogpt.NewClient(ApiKey),
		ctx:            ctx,
		userId:         UserId,
		maxQuestionLen: maxToken - maxAnswer, // 最大问题长度,问题长度尽量大(为了实现上下文会带着前面的问题)
		maxAnswerLen:   maxAnswer,            // 最大答案长度
		maxText:        maxToken,             // 最大文本 = 问题 + 回答, 接口限制
		timeOut:        timeOut,
		doneChan:       timeOutChan,
		cancel: func() {
			cancel()
		},
		ChatContext: NewContext(),
	}
}
func (c *ChatGPT) Close() {
	c.cancel()
}

func (c *ChatGPT) GetDoneChan() chan struct{} {
	return c.doneChan
}

func (c *ChatGPT) SetMaxQuestionLen(maxQuestionLen int) int {
	if maxQuestionLen > c.maxText-c.maxAnswerLen {
		maxQuestionLen = c.maxText - c.maxAnswerLen
	}
	c.maxQuestionLen = maxQuestionLen

	return c.maxQuestionLen
}

func (c *ChatGPT) Chat(question string) (answer string, err error) {
	question = question + "."
	if len(question) > c.maxQuestionLen {
		return "", OverMaxQuestionLength
	}
	if len(question)+c.maxAnswerLen > c.maxText {
		question = question[:c.maxText-c.maxAnswerLen]
	}
	req := gogpt.CompletionRequest{
		Model:            gogpt.GPT3TextDavinci003,
		MaxTokens:        c.maxAnswerLen,
		Prompt:           question,
		Temperature:      0.9,
		TopP:             1,
		N:                1,
		FrequencyPenalty: 0,
		PresencePenalty:  0.5,
		User:             c.userId,
		Stop:             []string{},
	}
	resp, err := c.client.CreateCompletion(c.ctx, req)
	if err != nil {
		return "", err
	}
	return formatAnswer(resp.Choices[0].Text), err
}
