package cli

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/reflection"
	"github.com/manifoldco/promptui"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"os"
	"sort"
)

const (
	actionTx    = "tx"
	actionQuery = "query"
)

func NewCLI(c *reflection.Client) *CLI {
	return &CLI{c: c}
}

type CLI struct {
	c *reflection.Client
}

func (p *CLI) Run() error {
	prompt := promptui.Select{
		Label: "Select action",
		Items: []string{actionTx, actionQuery},
	}

	_, res, err := prompt.Run()
	if err != nil {
		return err
	}

	switch res {
	case actionTx:
		return fmt.Errorf("not supported")
	case actionQuery:
		return p.query()
	default:
		return fmt.Errorf("unknown action: %s", res)
	}
}

func (p *CLI) query() error {
	m := p.c.ListQueryMap()

	selections := make([]string, 0, len(m))

	for k := range m {
		selections = append(selections, k)
	}

	sort.Slice(selections, func(i, j int) bool {
		return selections[i] < selections[j]
	})

	prompt := promptui.Select{
		Label: "method to query",
		Items: selections,
	}

	_, res, err := prompt.Run()
	if err != nil {
		return err
	}

	queryDesc, exists := m[res]
	if !exists {
		return fmt.Errorf("not found: %s", res)
	}

	dpb, err := fillDynamicMessagePrompt(queryDesc.Request)
	if err != nil {
		return err
	}

	resp, err := p.c.Query(context.TODO(), res, dpb)
	if err != nil {
		return err
	}

	b, err := p.c.Codec().MarshalJSON(resp)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stderr, fmt.Sprintf("%s", b))
	if err != nil {
		return err
	}

	return nil
}

func fillDynamicMessagePrompt(md protoreflect.MessageDescriptor) (*dynamicpb.Message, error) {
	dyn := dynamicpb.NewMessage(md)
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		v, err := valueFromFieldDescriptor(dyn, field)
		if err != nil {
			return nil, err
		}

		dyn.Set(field, v)
	}
	return dyn, nil
}

func valueFromFieldDescriptor(dyn *dynamicpb.Message, fd protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	label := fmt.Sprintf("fill field %s", fd.Name())
	switch fd.Kind() {
	case protoreflect.StringKind:
		prompt := promptui.Prompt{
			Label:   label,
			Default: fd.Default().String(),
		}

		res, err := prompt.Run()
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfString(res), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported kind: %s", fd.Kind())
	}
}