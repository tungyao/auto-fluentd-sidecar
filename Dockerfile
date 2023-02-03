FROM ubuntu
WORKDIR /home
COPY . .
RUN chmod +x /home/fluentd-side-crd
CMD ["/home/fluentd-side-crd"]