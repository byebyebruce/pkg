package chatgpt

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"strings"

	gogpt "github.com/sashabaranov/go-gpt3"
)

var (
	DefaultAiRole    = "AI"
	DefaultHumanRole = "Human"

	DefaultCharacter  = []string{"helpful", "creative", "clever", "friendly", "lovely", "talkative"}
	DefaultBackground = "The following is a conversation with AI assistant. The assistant is %s"
	DefaultPreset     = "\n%s: 你好，让我们开始愉快的谈话！\n%s: 我是 AI assistant ，请问你有什么问题？"
)

type (
	ChatContext struct {
		background  string // 对话背景
		preset      string // 预设对话
		maxSeqTimes int    // 最大对话次数
		aiRole      *role  // AI角色
		humanRole   *role  // 人类角色

		old        []conversation // 旧对话
		restartSeq string         // 重新开始对话的标识
		startSeq   string         // 开始对话的标识

		seqTimes int // 对话次数

		maintainSeqTimes bool // 是否维护对话次数 (自动移除旧对话)
	}

	ChatContextOption func(*ChatContext)

	conversation struct {
		Role   *role
		Prompt string
	}

	role struct {
		Name string
	}
)

func NewContext(options ...ChatContextOption) *ChatContext {
	ctx := &ChatContext{
		aiRole:           &role{Name: DefaultAiRole},
		humanRole:        &role{Name: DefaultHumanRole},
		background:       fmt.Sprintf(DefaultBackground, strings.Join(DefaultCharacter, ", ")+"."),
		maxSeqTimes:      1000,
		preset:           fmt.Sprintf(DefaultPreset, DefaultHumanRole, DefaultAiRole),
		old:              []conversation{},
		seqTimes:         0,
		restartSeq:       "\n" + DefaultHumanRole + ": ",
		startSeq:         "\n" + DefaultAiRole + ": ",
		maintainSeqTimes: false,
	}

	for _, option := range options {
		option(ctx)
	}
	return ctx
}

// PollConversation 移除最旧的一则对话
func (c *ChatContext) PollConversation() {
	c.old = c.old[1:]
	c.seqTimes--
}

// ResetConversation 重置对话
func (c *ChatContext) ResetConversation() {
	c.old = []conversation{}
	c.seqTimes = 0
}

// SaveConversation 保存对话
func (c *ChatContext) SaveConversation(path string) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(c.old)
	if err != nil {
		return err
	}
	return WriteToFile(path, buffer.Bytes())
}

// LoadConversation 加载对话
func (c *ChatContext) LoadConversation(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buffer)
	err = dec.Decode(&c.old)
	if err != nil {
		return err
	}
	c.seqTimes = len(c.old)
	return nil
}

func (c *ChatContext) SetHumanRole(role string) {
	c.humanRole.Name = role
	c.restartSeq = "\n" + c.humanRole.Name + ": "
}

func (c *ChatContext) SetAiRole(role string) {
	c.aiRole.Name = role
	c.startSeq = "\n" + c.aiRole.Name + ": "
}

func (c *ChatContext) SetMaxSeqTimes(times int) {
	c.maxSeqTimes = times
}

func (c *ChatContext) GetMaxSeqTimes() int {
	return c.maxSeqTimes
}

func (c *ChatContext) SetBackground(background string) {
	c.background = background
}

func (c *ChatContext) SetPreset(preset string) {
	c.preset = preset
}

func (c *ChatGPT) ChatWithContext(question string) (answer string, err error) {
	question = question + "."
	if len(question) > c.maxQuestionLen {
		return "", OverMaxQuestionLength
	}
	/*
		if c.ChatContext.seqTimes >= c.ChatContext.maxSeqTimes {
			if c.ChatContext.maintainSeqTimes {
				c.ChatContext.PollConversation()
			} else {
				return "", OverMaxSequenceTimes
			}
		}
	*/
	/*
		if len(c.ChatContext.old) > c.ChatContext.maxOld {
			c.ChatContext.PollConversation()
		}
	*/
	combinePrompt := func() string {
		var promptTable []string
		promptTable = append(promptTable, c.ChatContext.background)
		promptTable = append(promptTable, c.ChatContext.preset)
		for _, v := range c.ChatContext.old {
			if v.Role == c.ChatContext.humanRole {
				promptTable = append(promptTable, "\n"+v.Role.Name+": "+v.Prompt)
			} else {
				promptTable = append(promptTable, v.Role.Name+": "+v.Prompt)
			}
		}
		promptTable = append(promptTable, "\n"+c.ChatContext.restartSeq+question)
		prompt := strings.Join(promptTable, "\n")
		prompt += c.ChatContext.startSeq
		return prompt
	}
	/*
		var promptTable []string
		promptTable = append(promptTable, c.ChatContext.background)
		promptTable = append(promptTable, c.ChatContext.preset)
		for _, v := range c.ChatContext.old {
			if v.Role == c.ChatContext.humanRole {
				promptTable = append(promptTable, "\n"+v.Role.Name+": "+v.Prompt)
			} else {
				promptTable = append(promptTable, v.Role.Name+": "+v.Prompt)
			}
		}
		promptTable = append(promptTable, "\n"+c.ChatContext.restartSeq+question)
		prompt := strings.Join(promptTable, "\n")
		prompt += c.ChatContext.startSeq
	*/
	prompt := combinePrompt()
	for len(prompt) > c.maxText-c.maxAnswerLen {
		c.ChatContext.PollConversation()
		prompt = combinePrompt()
	}
	/*
		if len(prompt) > c.maxText-c.maxAnswerLen {
			return "", OverMaxTextLength
		}
	*/
	//error, status code: 400, message: This model's maximum context length is 4097 tokens, however you requested 4218 tokens (122 in your prompt; 4096 for the completion). Please reduce your prompt; or com
	req := gogpt.CompletionRequest{
		Model:            gogpt.GPT3TextDavinci003,
		MaxTokens:        c.maxAnswerLen,
		Prompt:           prompt,
		Temperature:      0.9,
		TopP:             1,
		N:                1,
		FrequencyPenalty: 0,
		PresencePenalty:  0.5,
		User:             c.userId,
		Stop:             []string{c.ChatContext.aiRole.Name + ":", c.ChatContext.humanRole.Name + ":"},
	}
	//fmt.Println("-------begin")
	//fmt.Println(prompt)
	//fmt.Println("-------end")
	resp, err := c.client.CreateCompletion(c.ctx, req)
	if err != nil {
		return "", err
	}
	resp.Choices[0].Text = formatAnswer(resp.Choices[0].Text)
	c.ChatContext.old = append(c.ChatContext.old, conversation{
		Role:   c.ChatContext.humanRole,
		Prompt: question,
	})
	c.ChatContext.old = append(c.ChatContext.old, conversation{
		Role:   c.ChatContext.aiRole,
		Prompt: resp.Choices[0].Text,
	})
	c.ChatContext.seqTimes++
	return resp.Choices[0].Text, nil
}

func WithMaxSeqTimes(times int) ChatContextOption {
	return func(c *ChatContext) {
		c.SetMaxSeqTimes(times)
	}
}

// WithOldConversation 从文件中加载对话
func WithOldConversation(path string) ChatContextOption {
	return func(c *ChatContext) {
		_ = c.LoadConversation(path)
	}
}

func WithMaintainSeqTimes(maintain bool) ChatContextOption {
	return func(c *ChatContext) {
		c.maintainSeqTimes = maintain
	}
}
