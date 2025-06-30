package doubao

import (
	"github.com/bytedance/sonic/ast"
	"github.com/labring/aiproxy/core/relay/meta"
)

func patchEmbeddingsVisionInput(node *ast.Node) error {
	inputNode := node.Get("input")
	if !inputNode.Exists() {
		return nil
	}

	switch inputNode.TypeSafe() {
	case ast.V_ARRAY:
		return inputNode.ForEach(func(_ ast.Sequence, item *ast.Node) bool {
			switch item.TypeSafe() {
			case ast.V_STRING:
				text, err := item.String()
				if err != nil {
					return false
				}

				*item = ast.NewObject([]ast.Pair{
					ast.NewPair("type", ast.NewString("text")),
					ast.NewPair("text", ast.NewString(text)),
				})

				return true
			case ast.V_OBJECT:
				textNode := item.Get("text")
				if textNode.Exists() && textNode.TypeSafe() == ast.V_STRING {
					_, err := item.Set("type", ast.NewString("text"))
					return err == nil
				}

				imageNode := item.Get("image")
				if imageNode.Exists() && imageNode.TypeSafe() == ast.V_STRING {
					imageURL, err := imageNode.String()
					if err != nil {
						return false
					}

					_, err = item.Unset("image")
					if err != nil {
						return false
					}

					_, err = item.Set("type", ast.NewString("image_url"))
					if err != nil {
						return false
					}

					_, err = item.SetAny("image_url", map[string]string{
						"url": imageURL,
					})
					if err != nil {
						return false
					}
				}

				return true
			default:
				return false
			}
		})
	case ast.V_STRING:
		inputText, err := inputNode.String()
		if err != nil {
			return err
		}

		_, err = node.SetAny("input", []map[string]string{
			{
				"type": "text",
				"text": inputText,
			},
		})

		return err
	default:
		return nil
	}
}

func embeddingPreHandler(_ *meta.Meta, node *ast.Node) error {
	return patchEmbeddingsVisionResponse(node)
}

func patchEmbeddingsVisionResponse(node *ast.Node) error {
	dataNode := node.Get("data")
	if !dataNode.Exists() {
		return nil
	}

	switch dataNode.TypeSafe() {
	case ast.V_ARRAY:
		return nil
	case ast.V_OBJECT:
		embeddingNode := dataNode.Get("embedding")
		if !embeddingNode.Exists() {
			return nil
		}

		_, err := node.Unset("data")
		if err != nil {
			return err
		}

		_, err = node.SetAny("data", []map[string]any{
			{
				"embedding": embeddingNode,
				"object":    "embedding",
				"index":     0,
			},
		})

		return err
	default:
		return nil
	}
}
