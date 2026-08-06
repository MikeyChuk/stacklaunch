import json
import logging
import os
import time
from typing import Any

import boto3
from botocore.exceptions import BotoCoreError, ClientError
from dotenv import load_dotenv


load_dotenv()

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s | %(levelname)s | %(message)s",
)

logger = logging.getLogger(__name__)

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


def validate_job(job: dict[str, Any]) -> None:
    required_fields = ["job_id", "job_type"]

    for field in required_fields:
        if field not in job:
            raise ValueError(f"Missing required field: {field}")


def send_welcome_email(job: dict[str, Any]) -> None:
    email = job.get("email")

    if not email:
        raise ValueError("email is required")

    logger.info(
        "Sending welcome email | job_id=%s email=%s",
        job["job_id"],
        email,
    )

    time.sleep(2)

    logger.info(
        "Welcome email sent | job_id=%s email=%s",
        job["job_id"],
        email,
    )


def process_job(job: dict[str, Any]) -> None:
    validate_job(job)

    job_type = job["job_type"]

    if job_type == "send_welcome_email":
        send_welcome_email(job)
        return

    raise ValueError(f"Unsupported job type: {job_type}")


def run_worker() -> None:
    logger.info("SQS worker started | queue_url=%s", QUEUE_URL)

    while True:
        try:
            response = sqs.receive_message(
                QueueUrl=QUEUE_URL,
                MaxNumberOfMessages=1,
                WaitTimeSeconds=20,
                VisibilityTimeout=30,
                AttributeNames=[
                    "ApproximateReceiveCount",
                ],
            )

            messages = response.get("Messages", [])

            if not messages:
                logger.info("No messages available")
                continue

            for message in messages:
                message_id = message["MessageId"]
                receipt_handle = message["ReceiptHandle"]

                receive_count = message.get(
                    "Attributes",
                    {},
                ).get(
                    "ApproximateReceiveCount",
                    "unknown",
                )

                logger.info(
                    "Message received | message_id=%s receive_count=%s",
                    message_id,
                    receive_count,
                )

                try:
                    job = json.loads(message["Body"])

                    process_job(job)

                    sqs.delete_message(
                        QueueUrl=QUEUE_URL,
                        ReceiptHandle=receipt_handle,
                    )

                    logger.info(
                        "Message deleted | message_id=%s job_id=%s",
                        message_id,
                        job.get("job_id"),
                    )

                except json.JSONDecodeError as error:
                    logger.error(
                        "Invalid JSON | message_id=%s error=%s",
                        message_id,
                        error,
                    )

                except Exception as error:
                    logger.exception(
                        "Job failed | message_id=%s error=%s",
                        message_id,
                        error,
                    )

        except (BotoCoreError, ClientError) as error:
            logger.exception("SQS communication failure: %s", error)
            time.sleep(5)


if __name__ == "__main__":
    try:
        run_worker()

    except KeyboardInterrupt:
        logger.info("Worker stopped")