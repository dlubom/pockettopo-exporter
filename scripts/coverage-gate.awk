# Use statement counts, not go tool cover's rounded percentage.
NR == 1 {
  if (NF != 2 || $1 != "mode:" || $2 !~ /^(set|count|atomic)$/) invalid = 1
  next
}
{
  if (NF < 3 || $(NF - 1) !~ /^[0-9]+$/ || $NF !~ /^[0-9]+$/) {
    invalid = 1
    next
  }
  statements = $(NF - 1)
  hits = $NF
  location = $0
  sub(/[ \t]+[0-9]+[ \t]+[0-9]+$/, "", location)
  # Match Go's merged blocks, including paths with spaces or drive letters.
  if (location in blocks) {
    if (blocks[location] != statements) invalid = 1
  } else {
    blocks[location] = statements
    total += statements
  }
  if (hits > 0 && !(location in coveredBlocks)) {
    coveredBlocks[location] = 1
    covered += statements
  }
}
END {
  if (invalid || total == 0) {
    print "Coverage gate requires a valid, nonempty statement profile." > "/dev/stderr"
    exit 1
  }
  if (100 * covered < 95 * total) {
    printf "Coverage below 95%%: %.0f of %.0f statements covered.\n", covered, total > "/dev/stderr"
    exit 1
  }
}
