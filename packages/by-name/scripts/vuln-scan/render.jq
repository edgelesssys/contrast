# Renders a list of findings (see normalize.jq) for humans.
#
# Args:
#   $format  "log" for one line per finding, "markdown" for a GitHub step summary table.
#   $title   heading of the markdown summary.

def pad($n): tostring | ($n - length) as $k | if $k > 0 then . + (" " * $k) else . end;

def truncate($n): if length > $n then .[:$n - 1] + "…" else . end;

def score_text: if .score == null then "-" else .score | tostring end;

def fixed_text: if .fixed == null then "-" else .fixed end;

def md_escape: gsub("\\|"; "\\|") | gsub("\n"; " ");

# Highest score first, findings without a score last.
def ordered: sort_by(-(.score // -1), .package, .id);

def log_lines:
  if length == 0 then "no findings"
  else
    ordered[]
    | [ (.tool | pad(6)),
        (.id | pad(19)),
        ("\(.package)@\(.version)" | pad(32)),
        (score_text | pad(4)),
        ("fix: \(fixed_text)" | pad(14)),
        (.title | truncate(100)),
        .link ]
    | join("  ")
  end;

def markdown:
  "### \($title)\n",
  if length == 0 then "No findings."
  else
    "| Score | ID | Package | Fixed in | Title | Tool |",
    "| ---: | --- | --- | --- | --- | --- |",
    (ordered[]
     | "| \(score_text) | [\(.id)](\(.link)) | `\(.package)@\(.version)` | \(fixed_text) | \(.title | truncate(200) | md_escape) | \(.tool) |")
  end,
  "";

if $format == "log" then log_lines
elif $format == "markdown" then markdown
else error("unknown format: \($format)") end
