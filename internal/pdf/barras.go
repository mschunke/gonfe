package pdf

import (
	"fmt"
	"strconv"
)

// padroesCode128 traz, para cada valor do Code 128, as larguras dos seis
// elementos que o compõem — barra, espaço, barra, espaço, barra, espaço —, em
// módulos. O último padrão, de parada, tem sete elementos.
//
// Todo padrão de dado soma exatamente onze módulos; o de parada soma treze. O
// teste do pacote confere essas duas invariantes em toda a tabela, o que
// transforma um erro de transcrição em falha de teste em vez de código de
// barras ilegível.
var padroesCode128 = [107]string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312",
	"132212", "221213", "221312", "231212", "112232", "122132", "122231", "113222",
	"123122", "123221", "223211", "221132", "221231", "213212", "223112", "312131",
	"311222", "321122", "321221", "312212", "322112", "322211", "212123", "212321",
	"232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121",
	"313121", "211331", "231131", "213113", "213311", "213131", "311123", "311321",
	"331121", "312113", "312311", "332111", "314111", "221411", "431111", "111224",
	"111422", "121124", "121421", "141122", "141221", "112214", "112412", "122114",
	"122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112",
	"421211", "212141", "214121", "412121", "111143", "111341", "131141", "114113",
	"114311", "411113", "411311", "113141", "114131", "311141", "411131", "211412",
	"211214", "211232", "2331112",
}

const (
	// inicioCode128C liga o modo C, que codifica dois dígitos por símbolo.
	inicioCode128C = 105
	// paradaCode128 encerra o código de barras.
	paradaCode128 = 106
)

// CodigoDeBarras desenha a chave de acesso em Code 128 modo C, o padrão que o
// DANFE exige.
//
// O modo C codifica pares de dígitos, então a entrada precisa ter uma
// quantidade par de dígitos — o que a chave de acesso, com 44, satisfaz.
func (pg *Pagina) CodigoDeBarras(posX, posY, largura, altura float64, digitos string) error {
	if len(digitos) == 0 {
		return fmt.Errorf("pdf: código de barras vazio")
	}
	if len(digitos)%2 != 0 {
		return fmt.Errorf("pdf: o Code 128 modo C exige quantidade par de dígitos; recebi %d", len(digitos))
	}

	valores := []int{inicioCode128C}
	for i := 0; i < len(digitos); i += 2 {
		par, err := strconv.Atoi(digitos[i : i+2])
		if err != nil {
			return fmt.Errorf("pdf: %q não é numérico: %w", digitos[i:i+2], err)
		}
		valores = append(valores, par)
	}

	// O dígito de controle é a soma ponderada dos valores, com peso igual à
	// posição a partir do símbolo de início, módulo 103.
	soma := inicioCode128C
	for i, v := range valores[1:] {
		soma += v * (i + 1)
	}
	valores = append(valores, soma%103, paradaCode128)

	// A largura de um módulo sai da largura total dividida pelo número de
	// módulos do código inteiro.
	modulos := 0
	for _, v := range valores {
		for _, d := range padroesCode128[v] {
			modulos += int(d - '0')
		}
	}
	larguraModulo := largura / float64(modulos)

	cursor := posX
	for _, v := range valores {
		barra := true
		for _, d := range padroesCode128[v] {
			espessura := float64(d-'0') * larguraModulo
			if barra {
				pg.RetanguloPreenchido(cursor, posY, espessura, altura, 0)
			}
			cursor += espessura
			barra = !barra
		}
	}
	return nil
}

// MatrizQR é um QR Code já codificado, em que cada posição verdadeira é um
// módulo escuro. A biblioteca não codifica QR Codes — ela produz o texto que
// vai dentro deles — e recebe a matriz pronta de quem imprime.
//
// Com github.com/skip2/go-qrcode, por exemplo:
//
//	q, _ := qrcode.New(nfe.InfNFeSupl.QrCode, qrcode.Medium)
//	matriz := pdf.MatrizQR(q.Bitmap())
type MatrizQR [][]bool

// Vazia informa se não há matriz a desenhar.
func (m MatrizQR) Vazia() bool { return len(m) == 0 || len(m[0]) == 0 }

// QRCode desenha a matriz como um quadrado de lado igual ao tamanho informado.
func (pg *Pagina) QRCode(posX, posY, tamanho float64, m MatrizQR) {
	if m.Vazia() {
		return
	}
	lado := len(m)
	modulo := tamanho / float64(lado)
	for linha := range m {
		for coluna := range m[linha] {
			if !m[linha][coluna] {
				continue
			}
			// Os módulos são desenhados com uma fração a mais de tamanho para
			// que não sobre um fio branco entre eles no arredondamento.
			pg.RetanguloPreenchido(
				posX+float64(coluna)*modulo,
				posY+float64(linha)*modulo,
				modulo*1.02, modulo*1.02, 0)
		}
	}
}
