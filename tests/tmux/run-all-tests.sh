#!/bin/bash
cd "$(dirname "$0")"
pass=0
fail=0
for test in test-*.sh; do
  bash "$test"
  if [ $? -eq 0 ]; then
    pass=$((pass+1))
  else
    fail=$((fail+1))
  fi
done
echo "\nSummary: $pass passed, $fail failed"
[ $fail -eq 0 ]