#!/bin/bash
BASE_DIR=`dirname $0`/../
TARGET_FILE=${BASE_DIR}templates/view.html
TMP_FILE=${TARGET_FILE}.tmp

echo '{{define "main.css"}}<style type="text/css">' > ${TARGET_FILE}
cat ${BASE_DIR}static/css/main.css >> ${TARGET_FILE}
echo '</style>{{end}}' >> ${TARGET_FILE}

echo '{{define "main.js"}}' >> ${TARGET_FILE}
find ${BASE_DIR}static/js -type f | while read FILE
do
  if [ ${FILE##*.} == js ] ; then
    echo '<script type="text/javascript">' >> ${TARGET_FILE}
    cat ${FILE} >> ${TARGET_FILE}
    echo '' >> ${TARGET_FILE}
    echo '</script>' >> ${TARGET_FILE}
  fi
done
echo '{{end}}' >> ${TARGET_FILE}

echo '{{define "favicon.ico"}}data:image/x-icon;base64,' >> ${TMP_FILE}
base64 -i ${BASE_DIR}static/img/favicon.ico >> ${TMP_FILE}
echo '{{end}}' >> ${TMP_FILE}

IMAGES="${BASE_DIR}static/img/*"
FILEARY=()
for FILEPATH in ${IMAGES}; do
  if [ -f ${FILEPATH} ] ; then
    FILEARY+=("${FILEPATH}")
  fi
done

for i in ${FILEARY[@]}; do
  FILENAME=`basename ${i}`
  if [ ${FILENAME##*.} == jpg ] ; then
    echo '{{define "'${FILENAME}'"}}data:image/jpeg;base64,' >> ${TMP_FILE}
    base64 -i ${i} >> ${TMP_FILE}
    echo '{{end}}' >> ${TMP_FILE}
  elif [ ${FILENAME##*.} == png ] ; then
    echo '{{define "'${FILENAME}'"}}data:image/png;base64,' >> ${TMP_FILE}
    base64 -i ${i} >> ${TMP_FILE}
    echo '{{end}}' >> ${TMP_FILE}
  fi
done

cat ${TMP_FILE} | awk '{printf $0}' >> ${TARGET_FILE}
rm ${TMP_FILE}
