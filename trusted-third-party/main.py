import pika

connection = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
channel = connection.channel()

# Declare the queue (replace 'my_queue' with your desired queue name)
queue_name = 'my_queue'
channel.queue_declare(queue=queue_name, durable=True)

# Bind the queue to the fanout exchange
exchange_name = 'block_events'
channel.queue_bind(exchange=exchange_name, queue=queue_name)

# Define callback function to process incoming messages
def callback(ch, method, properties, body):
    print("Received message:", body)

# Start consuming messages from the queue
try:
    channel.basic_consume(queue=queue_name, on_message_callback=callback, auto_ack=True)
    print('Waiting for messages. To exit press CTRL+C')
    channel.start_consuming()
except pika.exceptions.AMQPConnectionError as e:
    print("Error: Connection to RabbitMQ failed:", e)
except pika.exceptions.ProbableAuthenticationError as e:
    print("Error: Authentication failed:", e)
except pika.exceptions.ChannelClosedByBroker as e:
    print("Error: Channel closed by broker:", e)
except Exception as e:
    print("Error:", e)
finally:
    connection.close()

# Start consuming
print('Waiting for messages. To exit press CTRL+C')
channel.start_consuming()
