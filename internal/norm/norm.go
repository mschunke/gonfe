// Package norm normaliza estruturas de documentos fiscais antes da
// serialização, aplicando as regras declaradas em tags de campo.
//
// As tags reconhecidas são:
//
//	dec:"N"        reescala um tipos.Decimal para N casas decimais
//	norm:"upper"   converte a string para maiúsculas
//	norm:"num"     mantém apenas os dígitos da string
//	norm:"-"       deixa o campo intocado
//
// Toda string é, por padrão, aparada nas pontas, tem sequências de espaços em
// branco reduzidas a um único espaço e caracteres de controle removidos —
// exigências recorrentes dos validadores da SEFAZ, que rejeitam quebras de
// linha e espaços redundantes em campos de texto.
package norm

import (
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mschunke/gonfe/tipos"
)

var (
	tipoDecimal  = reflect.TypeOf(tipos.Decimal{})
	tipoDataHora = reflect.TypeOf(tipos.DataHora{})
	tipoData     = reflect.TypeOf(tipos.Data{})
	tipoTempo    = reflect.TypeOf(time.Time{})
)

// Normalizar percorre recursivamente v — que deve ser um ponteiro para struct —
// aplicando as regras das tags de cada campo. Campos não exportados, ponteiros
// nulos e tipos desconhecidos são ignorados.
func Normalizar(v any) {
	valor := reflect.ValueOf(v)
	if !valor.IsValid() {
		return
	}
	normalizarValor(valor, "")
}

func normalizarValor(v reflect.Value, tagDec string) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return
		}
		normalizarValor(v.Elem(), tagDec)

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			normalizarValor(v.Index(i), tagDec)
		}

	case reflect.String:
		if v.CanSet() {
			v.SetString(LimparTexto(v.String()))
		}

	case reflect.Struct:
		switch v.Type() {
		case tipoDecimal:
			aplicarEscala(v, tagDec)
			return
		case tipoDataHora, tipoData, tipoTempo:
			return
		}
		normalizarStruct(v)
	}
}

func normalizarStruct(v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		campo := t.Field(i)
		if !campo.IsExported() {
			continue
		}
		valorCampo := v.Field(i)
		regra := campo.Tag.Get("norm")
		if regra == "-" {
			continue
		}

		if valorCampo.Kind() == reflect.String && valorCampo.CanSet() {
			valorCampo.SetString(aplicarRegraTexto(valorCampo.String(), regra))
			continue
		}
		if valorCampo.Kind() == reflect.Pointer && !valorCampo.IsNil() &&
			valorCampo.Elem().Kind() == reflect.String && valorCampo.Elem().CanSet() {
			alvo := valorCampo.Elem()
			alvo.SetString(aplicarRegraTexto(alvo.String(), regra))
			continue
		}

		normalizarValor(valorCampo, campo.Tag.Get("dec"))
	}
}

func aplicarRegraTexto(s, regra string) string {
	s = LimparTexto(s)
	for _, r := range strings.Split(regra, ",") {
		switch strings.TrimSpace(r) {
		case "upper":
			s = strings.ToUpper(s)
		case "num":
			s = ApenasDigitos(s)
		}
	}
	return s
}

func aplicarEscala(v reflect.Value, tagDec string) {
	if tagDec == "" || !v.CanSet() {
		return
	}
	casas, err := strconv.ParseUint(tagDec, 10, 8)
	if err != nil || casas > tipos.CasasMax {
		return
	}
	d, ok := v.Interface().(tipos.Decimal)
	if !ok {
		return
	}
	v.Set(reflect.ValueOf(d.ComCasas(uint8(casas))))
}

// LimparTexto remove caracteres de controle, converte qualquer espaço em branco
// (inclusive tabulações e quebras de linha) em espaço simples, colapsa
// sequências de espaços e apara as pontas.
func LimparTexto(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	espacoPendente := false
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			espacoPendente = b.Len() > 0
		case r == '�' || (unicode.IsControl(r) && !unicode.IsSpace(r)):
			// Descarta caracteres de controle e o marcador de substituição.
		default:
			if espacoPendente {
				b.WriteByte(' ')
				espacoPendente = false
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ApenasDigitos devolve apenas os caracteres numéricos de s, descartando
// pontuação de máscaras como "12.345.678/0001-99".
func ApenasDigitos(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
