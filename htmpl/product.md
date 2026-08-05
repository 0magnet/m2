{{ $p := .Prod }}+++
URL = "/q/{{ $p.Partno }}"
Title = "{{ $p.Name }}"
Name = "{{ $p.Partno }}"
Categories = ["{{ $p.Category }}"]
+++
![{{ $p.Name }}](https://{{ .Domain }}/i/{{ $p.Category }}/{{ $p.Image1 }})
| | |
|:---|---:|
| | |
| Price: | ${{ $p.Price }} |<br>
| In stock: | {{ $p.Quantity }} |<br>
{{ if ne $p.Quantity "0" }}| Buy: | <my-button id="{{ $p.Partno }}"></my-button> |<br>{{ end }}
{{ if equalsIgnoreCase $p.Description1 $p.Name }}{{ else }}{{ if ne $p.Description1 "" }}| Description: | {{ safeHTML $p.Description1 }} |<br>{{ end }}{{ end }}
{{ if ne $p.Mfgname "" }}| Brand: | {{ $p.Mfgname }} |<br>{{ end }}
{{ if ne $p.Mfgpartno "" }}| MPN: | {{ $p.Mfgpartno }} |<br>{{ end }}
| Category: | {{ $p.Category }} |<br>
{{ if and (ne $p.VoltsRating "0") (ne $p.VoltsRating "0.0") (ne $p.VoltsRating "") }}| Voltage: | {{ $p.VoltsRating }} |<br>{{ end }}
{{ if and (ne $p.Value "0") (ne $p.Value "0.0") (ne $p.Value "") }}| Value: | {{ $p.Value }}{{ $p.ValUnit }} |<br>{{ end }}
{{ if and (ne $p.AmpsRating "0") (ne $p.AmpsRating "0.0") (ne $p.AmpsRating "") }}| Amperage: | {{ $p.AmpsRating }} |<br>{{ end }}
{{ if and (ne $p.Tolerance "0") (ne $p.Tolerance "") }}{{ $tolerancePercent := printf "%.2f%%" (mul 100 (toFloat $p.Tolerance)) }}| Tolerance: | {{ $tolerancePercent }} |<br>{{ end }}
{{ if ne $p.Typ "" }}| Type: | {{ $p.Typ }} |<br>{{ end }}
{{ if ne $p.Packagetype "" }}| Package Type: | {{ $p.Packagetype }}|<br>{{ end }}
{{ if ne $p.Technology "" }}| Technology: | {{ $p.Technology }} |<br>{{ end }}
{{ if ne $p.Materials "" }}| Materials: | {{ $p.Materials }} |<br>{{ end }}
{{ if and (ne $p.WattsRating "0") (ne $p.WattsRating "0.0") (ne $p.WattsRating "") }}| Watts Rating: | {{ $p.WattsRating }} |<br>{{ end }}
{{ if and (ne $p.Year "0") (ne $p.Year "") }}| Year: | {{ $p.Year }} |<br>{{ end }}
{{ if and (ne $p.CableLengthInches "0") (ne $p.CableLengthInches "0.0") (ne $p.CableLengthInches "") }}| Cable Length: | {{ $p.CableLengthInches }} inches |<br>{{ end }}
{{ if and (ne $p.WeightOz "0") (ne $p.WeightOz "0.0") }}| Weight: | {{ $p.WeightOz }} oz |<br>{{ end }}
{{ if and (ne $p.TempRating "0") (ne $p.TempRating "0.0") }}| Temp rating: | {{ $p.TempRating }}{{ $p.TempUnit }} |<br>{{ end }}
{{ if ne $p.Condition "" }}| Condition: | {{ $p.Condition }} |<br>{{ end }}
{{ if ne $p.Datasheet "" }}| Datasheet: | [{{ $p.Datasheet }}](https://{{ .Domain }}/i/pdf/{{$p.Datasheet}})|<br>{{ end }}
{{ if ne $p.Docs "" }}| Documentation: | {{ safeHTML  $p.Docs }} |<br>{{ end }}
{{ if ne $p.Note "" }}| Note: | {{ safeHTML  $p.Note }} |<br>{{ end }}
{{ if ne $p.Warning "" }}| Warning: | {{ safeHTML  $p.Warning }} |<br>{{ end }}
{{ if ne $p.Description2 "" }}| Additional Description: | {{ safeHTML  $p.Description2 }} |<br>{{ end }}
