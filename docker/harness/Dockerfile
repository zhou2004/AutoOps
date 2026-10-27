FROM python:3.12-slim
WORKDIR /opt/autoops-harness
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && useradd --uid 10001 --create-home harness
COPY bridge.py autoops-tools.mjs ./
USER 10001
EXPOSE 8091
CMD ["uvicorn", "bridge:app", "--host", "0.0.0.0", "--port", "8091", "--no-access-log"]
