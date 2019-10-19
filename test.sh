#!/usr/bin/env bash
BLACK=$'\e[30m' RED=$'\e[31m' GREEN=$'\e[32m' YELLOW=$'\e[33m' BLUE=$'\e[34m' PURPLE=$'\e[35m' LIGHT_BLUE=$'\e[36m' WHITE=$'\e[37m' GRAY=$'\e[90m' DEFAULT=$'\e[0m'

type >/dev/null 2>&1 icdiff && diff() {
  icdiff -W -U 1 --line-numbers "$@"
}
function run_test() {
  local target_dir=$1
  local all_cnt=0
  local passed_cnt=0
  local failed_cnt=0
  for input_filename in $(find "$target_dir" -name "*.groovy"); do
    output_filename=$(echo "$input_filename" | sed 's/input/output/g')
    tmp_output_filename="$output_filename.tmp.out"
    echo 1>&2 "# test of $input_filename"
    cat "$input_filename" | ./goenkins-format >"$tmp_output_filename"
    exit_code=$?

    if [[ $exit_code == 0 ]]; then
      echo 1>&2 "# test diff output"
      diff_output=$(command diff "$output_filename" "$tmp_output_filename")
      # echo $diff_output
      exit_code=$?
      if [[ $exit_code == 0 ]]; then
        echo 1>&2 "# [RESULT]: ${GREEN}PASS${DEFAULT}"
        ((passed_cnt++))
      else
        echo 1>&2 "# [RESULT]: ${RED}FAIL${DEFAULT}"
        # NOTE: icdiff exit_code is 0 even there is diff
        diff "$output_filename" "$tmp_output_filename"
        ((failed_cnt++))
      fi
    else
      echo 1>&2 "# [RESULT]: ${RED}FAIL${DEFAULT}"
      ((failed_cnt++))
    fi
    ((all_cnt++))
  done

  echo 1>&2 "# total"
  if [[ $passed_cnt == $all_cnt ]]; then
    echo 1>&2 "# [RESULT]: ${GREEN}PASS (passed/all)=($passed_cnt/$all_cnt)${DEFAULT} "
  else
    echo 1>&2 "# [RESULT]: ${RED}FAIL (passed/all)=($passed_cnt/$all_cnt)${DEFAULT}"
  fi
}
function main() {
  local target_dir=${1:-"test/input"}
  run_test "$target_dir"
}

main "$@"
