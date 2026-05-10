FROM nginx:stable-alpine

COPY ./html /usr/share/nginx/html
COPY ./nginx.conf /etc/nginx/nginx.conf 

RUN chmod 644 /usr/share/nginx/html/main.html && \
    chown nginx:nginx /usr/share/nginx/html/main.html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]