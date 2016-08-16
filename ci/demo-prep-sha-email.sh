#!/bin/bash
set -exu

SHA=$(shasum $file_to_hash | cut -d ' ' -f1)

output_body_file=email-out/$output_body_file
output_subject_file=email-out/$output_subject_file

echo -e "Email resource demo on $(date)" > $output_subject_file
echo -e "SHA1 Hash of \"$file_to_hash\" is $SHA\n\nCheers!\n" > $output_body_file
