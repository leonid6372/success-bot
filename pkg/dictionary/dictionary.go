package dictionary

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"

	"github.com/leonid6372/success-bot/pkg/format"
	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
)

const DefaultLanguage = "en"

type Dictionary struct {
	dictionary map[string]map[string]string // map[language_code]map[key]value

	digitSeparator   string
	decimalSeparator string
}

func New() (*Dictionary, error) {
	file, err := os.ReadFile("dictionary.json")
	if err != nil {
		return nil, err
	}

	var dictionary map[string]map[string]string
	if err := json.Unmarshal(file, &dictionary); err != nil {
		return nil, err
	}

	return &Dictionary{
		dictionary:       dictionary,
		digitSeparator:   " ",
		decimalSeparator: ",",
	}, nil
}

func (d *Dictionary) Languages() []string {
	langs := make([]string, 0, len(d.dictionary))

	for lang := range d.dictionary {
		langs = append(langs, lang)
	}

	return langs
}

func (d *Dictionary) Text(lang, key string, values ...map[string]any) string {
	text, ok := d.dictionary[lang][key]
	if !ok {
		log.Error("Text: value not found", zap.String("lang", lang), zap.String("key", key))
		return ""
	}

	tmpl, err := template.New(key).Parse(text)
	if err != nil {
		return text
	}

	var valuesMap map[string]any
	if len(values) > 0 {
		valuesMap = values[0]
	}

	// format numeric types in values
	for key, value := range valuesMap {
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			valuesMap[key] = format.PrettyNumber(v, d.digitSeparator, d.decimalSeparator)
		default:
			valuesMap[key] = value
		}
	}

	byteText := new(bytes.Buffer)
	if err = tmpl.Execute(byteText, valuesMap); err != nil {
		log.Error("Text: failed to execute template", zap.Error(err))
		return text
	}
	text = byteText.String()

	return text
}
