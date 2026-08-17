# Valores decimais

Nenhum valor fiscal desta biblioteca passa por `float64`. Essa é uma decisão de
projeto, não uma preferência estética.

## O problema com ponto flutuante

`float64` representa números em base 2. Valores como 0,1 e 0,07 não têm
representação exata nessa base, e o erro se acumula:

```go
soma := 0.0
for range 100 {
    soma += 0.07
}
fmt.Println(soma) // 7.000000000000005
```

Sete reais viram sete reais e uma fração invisível. Quando esse valor é
formatado com duas casas o erro some, mas quando ele participa de uma
comparação, de uma diferença ou de mais uma multiplicação, o centavo aparece —
e a SEFAZ rejeita a nota por soma que não fecha.

Com [`tipos.Decimal`](https://pkg.go.dev/github.com/mschunke/gonfe/tipos#Decimal)
o mesmo laço dá exatamente `7.00`.

## Como funciona

`Decimal` guarda um inteiro sem escala e a quantidade de casas decimais. O valor
`10,50` é o par `(1050, 2)`. Todas as operações são feitas em inteiros, então
não há erro de representação.

```go
a := tipos.D("10.50")       // 1050 com 2 casas
b := tipos.D("0.255")       // 255 com 3 casas

a.Somar(b)                  // 10.755  — assume a maior escala
a.Subtrair(b)               // 10.245
a.Multiplicar(b)            // 2.67750 — escalas se somam
a.MultiplicarCom(b, 2)      // 2.68    — arredondado para 2 casas
a.Percentual(tipos.D("18.00"), 2) // 1.89 — 18% de 10,50
```

## Construindo valores

| Função | Uso |
| --- | --- |
| `tipos.D("199.90")` | Literais no código; entra em pânico se o texto for inválido |
| `tipos.ParseDecimal(s)` | Entrada externa, devolve erro |
| `tipos.ParseDecimalBR(s)` | Formato brasileiro: `"1.234,56"` |
| `tipos.NovoDecimal(19990, 2)` | A partir de centavos, quando o banco de dados guarda inteiros |
| `tipos.DeInteiro(42)` | Inteiro sem casas decimais |

A escala é preservada exatamente como escrita: `tipos.D("10.50")` tem duas
casas, `tipos.D("10.5")` tem uma, e as duas são numericamente iguais para
`Igual` e `Comparar`.

!!! warning "Nunca converta de `float64`"

    Não existe construtor a partir de `float64`, e isso é de propósito. Se o
    valor chega como `float64` de um sistema que você não controla, formate-o
    com a precisão desejada antes de converter:

    ```go
    d, err := tipos.ParseDecimal(strconv.FormatFloat(v, 'f', 2, 64))
    ```

    `Float64()` existe para o caminho inverso — exibição e cálculos aproximados
    — e nunca deve alimentar um XML fiscal.

## Escala automática pelo leiaute

Cada campo do leiaute tem uma precisão obrigatória: `vProd` tem duas casas,
`qCom` tem quatro, `vUnCom` tem dez. Escrever `<vProd>25</vProd>` é rejeição na
certa; o correto é `<vProd>25.00</vProd>`.

Você não precisa se preocupar com isso. As estruturas do pacote `nfe` declaram a
precisão de cada campo em uma tag, e `NFe.Preparar` reescala tudo antes de
serializar:

```go
type Prod struct {
    QCom   tipos.Decimal `xml:"qCom" dec:"4"`
    VUnCom tipos.Decimal `xml:"vUnCom" dec:"10"`
    VProd  tipos.Decimal `xml:"vProd" dec:"2"`
}
```

```go
prod.QCom = tipos.D("10")   // você escreve assim
// depois de Preparar, o XML sai <qCom>10.0000</qCom>
```

## Arredondamento

O arredondamento é **comercial**: metade para longe do zero, que é o critério
adotado pelos manuais da SEFAZ.

```go
tipos.D("2.345").ComCasas(2)  // 2.35
tipos.D("2.344").ComCasas(2)  // 2.34
tipos.D("-2.345").ComCasas(2) // -2.35
tipos.D("10.5").ComCasas(0)   // 11
```

Isso difere do arredondamento bancário (metade para o par) usado por várias
linguagens, e da truncagem. Se o seu ERP arredonda de outro jeito, os totais
podem divergir em um centavo — a validação da biblioteca tolera essa diferença
por item, como o próprio leiaute faz.

## Arredondar por item, somar depois

A ordem das operações importa e é fixada pelo leiaute: o tributo é calculado e
arredondado **por item**, e o total é a soma dos valores já arredondados.

```go
// Certo: arredonda cada item, depois soma.
item1 := tipos.D("25.00").Percentual(tipos.D("1.65"), 2)  // 0.41
item2 := tipos.D("74.70").Percentual(tipos.D("1.65"), 2)  // 1.23
total := item1.Somar(item2)                               // 1.64

// Errado: soma as bases e arredonda no fim.
tipos.D("99.70").Percentual(tipos.D("1.65"), 2)           // 1.65
```

Um centavo de diferença. `CalcularTotais` faz do jeito certo.

## Serialização

`Decimal` implementa as interfaces de XML e de JSON. Em JSON o valor sai como
string, para não perder precisão ao passar por um parser que use `float64`:

```json
{"vProd": "1250.00", "qCom": "10.0000"}
```

Na leitura, tanto string quanto número JSON são aceitos.
