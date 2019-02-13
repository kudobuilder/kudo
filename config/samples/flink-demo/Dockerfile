FROM nginx

RUN apt-get update && apt-get install -y curl wget jq

ADD upload.sh /



CMD ["/upload.sh"]