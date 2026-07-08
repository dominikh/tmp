#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
Usage: ./plot-y-over-repetitions.sh INPUT.csv [OUTPUT.png] [average|boxplot] [--outliers]

Reads CSV columns: x,repetitions,y. A header row is optional.
Plots y/repetitions grouped by x.

Modes:
  average   Line plot of average y/repetitions per x. Default.
  boxplot   One box plot per x.

The output image width is chosen from the number of distinct x values:
18/8 pixels per x value, with a minimum width of 800 pixels.

The X axis has a tick at each integer value, but labels only multiples of 32.
EOF
  exit 2
}

[[ $# -ge 1 ]] || usage

infile=$1
shift
outfile="plot.png"
mode="average"
outlier_mode="nooutliers"

# Optional output file comes first, unless the next argument is clearly an option/mode.
if [[ $# -gt 0 && $1 != "average" && $1 != "minavgmax" && $1 != "boxplot" && $1 != --* ]]; then
  outfile=$1
  shift
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    average|minavgmax)
      # minavgmax is accepted as a backwards-compatible alias for average.
      mode="average"
      ;;
    boxplot)
      mode="boxplot"
      ;;
    --outliers)
      mode="boxplot"
      outlier_mode="outliers pointtype 7"
      ;;
    --no-outliers)
      outlier_mode="nooutliers"
      ;;
    *)
      usage
      ;;
  esac
  shift
done

data=$(mktemp)
summary=$(mktemp)
trap 'rm -f "$data" "$summary"' EXIT

# Convert CSV to tab-separated: x, y/repetitions.
awk -F, '
function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
function isnum(s) { return s ~ /^[-+]?([0-9]+\.?[0-9]*|\.[0-9]+)([eE][-+]?[0-9]+)?$/ }
BEGIN { OFS="\t" }
NR == 1 {
  first = trim($1)
  if (tolower(first) == "x" || !isnum(first)) next
}
NF >= 3 {
  x = trim($1)
  repetitions = trim($2) + 0
  y = trim($3) + 0
  if (repetitions == 0) next
  print x, y / repetitions
}
' "$infile" | sort -n -k1,1 -k2,2 > "$data"

num_x=$(awk '{ seen[$1] = 1 } END { print length(seen) }' "$data")
if (( num_x == 0 )); then
  echo "No data rows found in $infile" >&2
  exit 1
fi

# Summary columns: x, avg, n.
awk '
BEGIN { OFS="\t" }
{
  x = $1
  v = $2 + 0
  n[x]++
  sum[x] += v
}
END {
  for (x in n)
    print x, sum[x] / n[x], n[x]
}
' "$data" | sort -n -k1,1 > "$summary"

width=$(((num_x * 18 + 7) / 8))
if (( width < 800 )); then
  width=800
fi
height=700

if [[ $mode == "average" ]]; then
  gnuplot <<GP
set terminal pngcairo size ${width},${height} enhanced font ",12"
set output "$outfile"
set datafile separator "\t"
set xlabel "Vector length"
set ylabel "ns"
set title "Average time per Pop"
set grid ytics
set key off
set yrange [0:*]
set xtics 32 nomirror
set mxtics 32
set format x "%.0f"
set offset graph 0.01, graph 0.01, graph 0.05, 0
stats "$summary" using 1 nooutput
set xrange [STATS_min - 0.5 : STATS_max + 0.5]

plot "$summary" using 1:2 with lines \
       linewidth 2.5 linecolor rgb "#1f77b4"
GP
else
  gnuplot <<GP
set terminal pngcairo size ${width},${height} enhanced font ",12"
set output "$outfile"
set datafile separator "\t"
set xlabel "Vector length"
set ylabel "ns"
set title "Average time per Pop"
set grid ytics
set key off
set clip points
set xtics 32 nomirror
set mxtics 32
set format x "%.0f"
stats "$data" using 1 nooutput
set xrange [STATS_min - 0.5 : STATS_max + 0.5]

set style data boxplot
set style boxplot $outlier_mode labels off
set style fill solid 0.5 border rgb "#1f77b4"
set boxwidth 0.5
set bars 0.5

# Use x as the boxplot factor, so each distinct x gets one box.
plot "$data" using (1):2:(0.5):1 with boxplot linecolor rgb "#1f77b4"
GP
fi

echo "Wrote $outfile"
