Add-Type -AssemblyName System.Drawing

$size = 1024
$bitmap = [System.Drawing.Bitmap]::new($size, $size)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$graphics.Clear([System.Drawing.Color]::Transparent)

function RoundedPath([float]$x, [float]$y, [float]$width, [float]$height, [float]$radius) {
    $path = [System.Drawing.Drawing2D.GraphicsPath]::new()
    $diameter = $radius * 2
    $path.AddArc($x, $y, $diameter, $diameter, 180, 90)
    $path.AddArc($x + $width - $diameter, $y, $diameter, $diameter, 270, 90)
    $path.AddArc($x + $width - $diameter, $y + $height - $diameter, $diameter, $diameter, 0, 90)
    $path.AddArc($x, $y + $height - $diameter, $diameter, $diameter, 90, 90)
    $path.CloseFigure()
    return $path
}

$outer = RoundedPath 54 54 916 916 212
$navy = [System.Drawing.Drawing2D.LinearGradientBrush]::new([System.Drawing.Point]::new(0, 0), [System.Drawing.Point]::new(1024, 1024), [System.Drawing.ColorTranslator]::FromHtml('#203b5b'), [System.Drawing.ColorTranslator]::FromHtml('#091526'))
$graphics.FillPath($navy, $outer)
$border = [System.Drawing.Pen]::new([System.Drawing.ColorTranslator]::FromHtml('#415875'), 10)
$graphics.DrawPath($border, $outer)

$halo = [System.Drawing.Pen]::new([System.Drawing.Color]::FromArgb(40, 227, 193, 122), 7)
$graphics.DrawEllipse($halo, 188, 188, 648, 648)

$tile = RoundedPath 258 258 508 508 130
$gold = [System.Drawing.Drawing2D.LinearGradientBrush]::new([System.Drawing.Point]::new(285, 280), [System.Drawing.Point]::new(740, 750), [System.Drawing.ColorTranslator]::FromHtml('#f3d998'), [System.Drawing.ColorTranslator]::FromHtml('#b47c31'))
$graphics.FillPath($gold, $tile)

$ink = [System.Drawing.ColorTranslator]::FromHtml('#142842')
$shaft = [System.Drawing.Pen]::new($ink, 56)
$shaft.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
$shaft.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
$shaft.LineJoin = [System.Drawing.Drawing2D.LineJoin]::Round
$graphics.DrawLine($shaft, 512, 373, 512, 569)
$graphics.DrawLines($shaft, [System.Drawing.Point[]]@([System.Drawing.Point]::new(409, 491), [System.Drawing.Point]::new(512, 594), [System.Drawing.Point]::new(615, 491)))
$graphics.DrawLine($shaft, 390, 652, 634, 652)

$outPath = Join-Path $PSScriptRoot '..\build\appicon.png'
$bitmap.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)

$shaft.Dispose(); $gold.Dispose(); $tile.Dispose(); $halo.Dispose(); $border.Dispose(); $navy.Dispose(); $outer.Dispose(); $graphics.Dispose(); $bitmap.Dispose()
