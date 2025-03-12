package python

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"globstar.dev/analysis"
)

var DjangoInsecureEval *analysis.Analyzer = &analysis.Analyzer{
	Name:        "django-insecure-eval",
	Language:    analysis.LangPy,
	Description: "Using `eval` with user data creates a severe security vulnerability that allows attackers to execute arbitrary code on your system. This dangerous practice can lead to complete system compromise, data theft, or service disruption. Instead, replace `eval` with dedicated libraries or methods specifically designed for your required functionality.",
	Category:    analysis.CategorySecurity,
	Severity:    analysis.SeverityWarning,
	Run:         checkDjangoInsecureEval,
}

func checkDjangoInsecureEval(pass *analysis.Pass) (interface{}, error) {
	requestVarMap := make(map[string]bool)
	userFmtStrVarMap := make(map[string]bool)

	// first pass: check for assignment of `request` data stored in variables
	analysis.Preorder(pass, func(node *sitter.Node) {
		if node.Type() != "assignment" {
			return
		}

		leftNode := node.ChildByFieldName("left")
		rightNode := node.ChildByFieldName("right")

		if rightNode.Type() != "call" && rightNode.Type() != "subscript" && rightNode.Type() != "binary_operator" {
			return
		}

		if isRequestCall(rightNode, pass.FileContext.Source) {
			varName := leftNode.Content(pass.FileContext.Source)
			requestVarMap[varName] = true
		}

	})

	analysis.Preorder(pass, func(node *sitter.Node) {
		if node.Type() != "assignment" {
			return
		}

		leftNode := node.ChildByFieldName("left")
		rightNode := node.ChildByFieldName("right")

		if isStringFormatted(rightNode, pass.FileContext.Source, requestVarMap) {
			userFmtStrVarMap[leftNode.Content(pass.FileContext.Source)] = true
		}

	})

	analysis.Preorder(pass, func(node *sitter.Node) {
		if node.Type() != "call" {
			return
		}

		funcNode := node.ChildByFieldName("function")
		if !strings.Contains(funcNode.Content(pass.FileContext.Source), "eval") {
			return
		}

		argNode := node.ChildByFieldName("arguments")

		argumentList := getNamedChildren(argNode, 0)

		for _, arg := range argumentList {
			if arg.Type() == "identifier" {
				// check for `request` method call var
				for key := range requestVarMap {
					if key == arg.Content(pass.FileContext.Source) {
						pass.Report(pass, node, "Detected user data in `eval` call which can cause remote code execution")
					}
				}

				// check for string user data formatted string var
				for key := range userFmtStrVarMap {
					if key == arg.Content(pass.FileContext.Source) {
						pass.Report(pass, node, "Detected user data in `eval` call which can cause remote code execution")
					}
				}
			} else if isRequestCall(arg, pass.FileContext.Source) {
				pass.Report(pass, node, "Detected user data in `eval` call which can cause remote code execution")
			} else if arg.Type() == "binary_operator" {
				rightNode := arg.ChildByFieldName("right")
				if isRequestCall(rightNode, pass.FileContext.Source) {
					pass.Report(pass, node, "Detected user data in `eval` call which can cause remote code execution")
				}
			} else if isStringFormatted(arg, pass.FileContext.Source, requestVarMap) {
				pass.Report(pass, node, "Detected user data in `eval` call which can cause remote code execution")
			}
		}

	})
	return nil, nil
}

func isStringFormatted(node *sitter.Node, source []byte, reqVarMap map[string]bool) bool {
	switch node.Type() {
	case "call":
		funcNode := node.ChildByFieldName("function")
		if funcNode.Type() != "attribute" {
			return false
		}
		strObjectNode := funcNode.ChildByFieldName("object")
		funcAttribute := funcNode.Content(source)
		if !strings.HasSuffix(funcAttribute, ".format") && strObjectNode.Type() != "string" {
			return false
		}

		argNode := node.ChildByFieldName("arguments")
		if argNode.Type() != "argument_list" {
			return false
		}

		reqArgNode := argNode.NamedChild(0)
		if !isRequestCall(reqArgNode, source) && !hasUserDataVar(reqArgNode, source, reqVarMap) {
			return false
		}

		return true

	case "binary_operator":
		binaryOpLeftNode := node.ChildByFieldName("left")
		binaryOpRightNode := node.ChildByFieldName("right")
		if binaryOpLeftNode.Type() != "string" {
			return false
		}

		if !isRequestCall(binaryOpRightNode, source) && !hasUserDataVar(binaryOpRightNode, source, reqVarMap) {
			return false
		}

		return true

	case "string":
		strContent := node.Content(source)
		// check if f-string
		if strContent[0] != 'f' {
			return false
		}

		allChildren := getNamedChildren(node, 0)

		// check if user data is present in f-string interpolation
		for _, child := range allChildren {
			if child.Type() == "interpolation" {
				if isRequestCall(child.NamedChild(0), source) || hasUserDataVar(child.NamedChild(0), source, reqVarMap) {
					return true
				}
			}
		}

	}

	return false
}

func hasUserDataVar(node *sitter.Node, source []byte, reqVarMap map[string]bool) bool {
	if node.Type() != "identifier" {
		return false
	}

	argName := node.Content(source)

	for key := range reqVarMap {
		if argName == key {
			return true
		}
	}

	return false
}

func isRequestCall(node *sitter.Node, source []byte) bool {
	switch node.Type() {
	case "call":
		funcNode := node.ChildByFieldName("function")
		if funcNode.Type() != "attribute" {
			return false
		}
		objectNode := funcNode.ChildByFieldName("object")
		if !strings.Contains(objectNode.Content(source), "request") {
			return false
		}

		attributeNode := funcNode.ChildByFieldName("attribute")
		if attributeNode.Type() != "identifier" {
			return false
		}

		if !strings.Contains(attributeNode.Content(source), "get") {
			return false
		}

		return true

	case "subscript":
		valueNode := node.ChildByFieldName("value")
		if valueNode.Type() != "attribute" {
			return false
		}

		objNode := valueNode.ChildByFieldName("object")
		if objNode.Type() != "identifier" && objNode.Content(source) != "request" {
			return false
		}

		return true
	}

	return false
}
