just coordinator

sed -i -e 's/kata-qemu-snp/contrast-cc-81ebecbf2bde082dedd98b337fbbd386/g' workspace/deployment/deployment.yml

just generate

sed -i -e 's/contrast-cc-81ebecbf2bde082dedd98b337fbbd386/kata-qemu-snp/g' workspace/deployment/deployment.yml

kubectl apply -f workspace/deployment/

sed -i -e 's/kata-qemu-snp/contrast-cc-81ebecbf2bde082dedd98b337fbbd386/g' workspace/deployment/deployment.yml

sleep 5
just set
