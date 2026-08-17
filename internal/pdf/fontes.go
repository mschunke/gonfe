package pdf

import "strings"

// As larguras abaixo são as das fontes base-14, em milésimos do corpo, como
// publicadas nos arquivos de métrica da Adobe. Sem elas não há como centralizar
// nem cortar texto para caber em uma caixa.

// larguraHelvetica mapeia os caracteres imprimíveis de ASCII para sua largura.
var larguraHelvetica = map[rune]int{
	' ': 278, '!': 278, '"': 355, '#': 556, '$': 556, '%': 889, '&': 667, '\'': 191,
	'(': 333, ')': 333, '*': 389, '+': 584, ',': 278, '-': 333, '.': 278, '/': 278,
	'0': 556, '1': 556, '2': 556, '3': 556, '4': 556, '5': 556, '6': 556, '7': 556,
	'8': 556, '9': 556, ':': 278, ';': 278, '<': 584, '=': 584, '>': 584, '?': 556,
	'@': 1015,
	'A': 667, 'B': 667, 'C': 722, 'D': 722, 'E': 667, 'F': 611, 'G': 778, 'H': 722,
	'I': 278, 'J': 500, 'K': 667, 'L': 556, 'M': 833, 'N': 722, 'O': 778, 'P': 667,
	'Q': 778, 'R': 722, 'S': 667, 'T': 611, 'U': 722, 'V': 667, 'W': 944, 'X': 667,
	'Y': 667, 'Z': 611,
	'[': 278, '\\': 278, ']': 278, '^': 469, '_': 556, '`': 333,
	'a': 556, 'b': 556, 'c': 500, 'd': 556, 'e': 556, 'f': 278, 'g': 556, 'h': 556,
	'i': 222, 'j': 222, 'k': 500, 'l': 222, 'm': 833, 'n': 556, 'o': 556, 'p': 556,
	'q': 556, 'r': 333, 's': 500, 't': 278, 'u': 556, 'v': 500, 'w': 722, 'x': 500,
	'y': 500, 'z': 500,
	'{': 334, '|': 260, '}': 334, '~': 584,
}

// larguraHelveticaNegrito é a métrica da Helvetica-Bold.
var larguraHelveticaNegrito = map[rune]int{
	' ': 278, '!': 333, '"': 474, '#': 556, '$': 556, '%': 889, '&': 722, '\'': 238,
	'(': 333, ')': 333, '*': 389, '+': 584, ',': 278, '-': 333, '.': 278, '/': 278,
	'0': 556, '1': 556, '2': 556, '3': 556, '4': 556, '5': 556, '6': 556, '7': 556,
	'8': 556, '9': 556, ':': 333, ';': 333, '<': 584, '=': 584, '>': 584, '?': 611,
	'@': 975,
	'A': 722, 'B': 722, 'C': 722, 'D': 722, 'E': 667, 'F': 611, 'G': 778, 'H': 722,
	'I': 278, 'J': 556, 'K': 722, 'L': 611, 'M': 833, 'N': 722, 'O': 778, 'P': 667,
	'Q': 778, 'R': 722, 'S': 667, 'T': 611, 'U': 722, 'V': 667, 'W': 944, 'X': 667,
	'Y': 667, 'Z': 611,
	'[': 333, '\\': 278, ']': 333, '^': 584, '_': 556, '`': 333,
	'a': 556, 'b': 611, 'c': 556, 'd': 611, 'e': 556, 'f': 333, 'g': 611, 'h': 611,
	'i': 278, 'j': 278, 'k': 556, 'l': 278, 'm': 889, 'n': 611, 'o': 611, 'p': 611,
	'q': 611, 'r': 389, 's': 556, 't': 333, 'u': 611, 'v': 556, 'w': 778, 'x': 556,
	'y': 556, 'z': 500,
	'{': 389, '|': 280, '}': 389, '~': 584,
}

// semAcento devolve a letra latina básica correspondente a um caractere
// acentuado. As fontes base-14 dão ao caractere acentuado a mesma largura da
// letra que ele acentua, então medir pela base é exato.
var semAcento = map[rune]rune{
	'À': 'A', 'Á': 'A', 'Â': 'A', 'Ã': 'A', 'Ä': 'A', 'Å': 'A',
	'È': 'E', 'É': 'E', 'Ê': 'E', 'Ë': 'E',
	'Ì': 'I', 'Í': 'I', 'Î': 'I', 'Ï': 'I',
	'Ò': 'O', 'Ó': 'O', 'Ô': 'O', 'Õ': 'O', 'Ö': 'O',
	'Ù': 'U', 'Ú': 'U', 'Û': 'U', 'Ü': 'U',
	'Ç': 'C', 'Ñ': 'N', 'Ý': 'Y',
	'à': 'a', 'á': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a',
	'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e',
	'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i',
	'ò': 'o', 'ó': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n', 'ý': 'y', 'ÿ': 'y',
}

// larguraDoCaractere devolve a largura de um caractere em milésimos do corpo.
func larguraDoCaractere(r rune, f Fonte) int {
	// A Courier é monoespaçada: todo caractere tem a mesma largura.
	if f == Monoespacada {
		return 600
	}
	tabela := larguraHelvetica
	if f == Negrito {
		tabela = larguraHelveticaNegrito
	}
	if l, ok := tabela[r]; ok {
		return l
	}
	if base, ok := semAcento[r]; ok {
		if l, ok := tabela[base]; ok {
			return l
		}
	}
	switch r {
	case 'º', 'ª':
		return 365
	case '°':
		return 400
	case '§':
		return 556
	case '…':
		return 1000
	}
	// Um caractere desconhecido vira "?" na conversão para WinAnsi, então medir
	// como "?" mantém a largura coerente com o que será desenhado.
	return tabela['?']
}

// winAnsiEspeciais mapeia os caracteres que o Windows-1252 posiciona na faixa
// 0x80–0x9F, onde o Latin-1 não define nada imprimível.
var winAnsiEspeciais = map[rune]byte{
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87,
	'ˆ': 0x88, '‰': 0x89, 'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C, 'Ž': 0x8E,
	'‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95, '–': 0x96, '—': 0x97,
	'˜': 0x98, '™': 0x99, 'š': 0x9A, '›': 0x9B, 'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F,
}

// paraWinAnsi converte texto UTF-8 para a codificação WinAnsi declarada nas
// fontes do documento. Caracteres fora dela viram "?", o que é preferível a
// produzir um PDF ilegível.
func paraWinAnsi(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r < 0x80:
			b.WriteByte(byte(r))
		case r <= 0xFF:
			b.WriteByte(byte(r))
		default:
			if c, ok := winAnsiEspeciais[r]; ok {
				b.WriteByte(c)
				continue
			}
			if base, ok := semAcento[r]; ok {
				b.WriteByte(byte(base))
				continue
			}
			b.WriteByte('?')
		}
	}
	return b.String()
}
