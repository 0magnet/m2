+++
URL   = "/c/{{ .Category }}"
Title = "{{ if eq .Category "" }}All Products{{ else }}Category: {{ .Category }}{{ end }}"
Name  = "{{ if eq .Category "" }}All Products{{ else }}{{ .Category }}{{ end }}"
+++

| # | Item | Price | Stock | Buy |
|:-:|:-----|------:|------:|:---:|
{{- range $i, $p := .Products }}
| {{ add $i 1 }} | [{{ $p.Name }}](/q/{{ $p.Partno }}) | ${{ $p.Price }} | {{ $p.Quantity }} | {{ if ne $p.Quantity "0" }}<my-button id='{{ $p.Partno }}'>Add to cart</my-button>{{ else }}out of stock{{ end }} |
{{- end }}
