import json
import os
import uuid

import boto3
from botocore.exceptions import BotoCoreError, ClientError
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException, status
from pydantic import BaseModel, EmailStr


load_dotenv()

app = FastAPI(
    title="StackLaunch SQS Producer",
    version="1.0.0",
)

AWS_REGION = os.getenv("AWS_REGION")
QUEUE_URL = os.getenv("QUEUE_URL")

if not AWS_REGION:
    raise RuntimeError("AWS_REGION is not configured")

if not QUEUE_URL:
    raise RuntimeError("QUEUE_URL is not configured")

sqs = boto3.client(
    "sqs",
    region_name=AWS_REGION,
)


class CreateUserRequest(BaseModel):
    email: EmailStr


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "healthy"}


@app.post(
    "/users",
    status_code=status.HTTP_202_ACCEPTED,
)
def create_user(request: CreateUserRequest) -> dict[str, str]:
    job = {
        "job_id": str(uuid.uuid4()),
        "job_type": "send_welcome_email",
        "email": request.email,
    }

    try:
        response = sqs.send_message(
            QueueUrl=QUEUE_URL,
            MessageBody=json.dumps(job),
        )

    except (BotoCoreError, ClientError) as error:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Unable to queue the background job",
        ) from error

    return {
        "status": "queued",
        "job_id": job["job_id"],
        "message_id": response["MessageId"],
    }