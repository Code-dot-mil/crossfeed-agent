output="output/portscan/sonar/$2"
wget https://opendata.rapid7.com/sonar.tcp/$1.csv.gz -O "$output.csv.gz"
gunzip -k "$output".csv.gz
awk -F "\"*,\"*" '{print $2}' "$output".csv > "$output"-ips.txt
sort -o "$output".txt "$output"-ips.txt
rm "$output.csv.gz"
rm "$output.csv"
rm "$output-ips.txt"
