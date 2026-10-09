# Do not count missing, invalid, uncovered or timed-out mutants as killed.
[.files[].mutations[]] as $mutants |
($mutants | length) > 0 and
($mutants | length) == .mutants_total and
all($mutants[]; .status == "KILLED" or .status == "LIVED") and
([ $mutants[] | select(.status == "KILLED") ] | length) / ($mutants | length) >= 0.9
