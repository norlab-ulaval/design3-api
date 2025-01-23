FROM python:3.12-alpine as dev

EXPOSE 5000

WORKDIR /app

COPY requirements.txt .
RUN pip3 install -r requirements.txt

COPY . .

ENTRYPOINT ["python3"]
CMD ["app.py"]


FROM dev as prod

ENTRYPOINT ["waitress-serve", "--listen", "0.0.0.0:5000", "--call", "app:create_app"]