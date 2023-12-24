import os
import psycopg2

# Obtain database credentials and table names from environment variables
DB_NAME = os.environ.get('DB_NAME')
DB_USER = os.environ.get('DB_USER')
DB_PASSWORD = os.environ.get('DB_PASSWORD')
DB_HOST = os.environ.get('DB_HOST')
DB_PORT = os.environ.get('DB_PORT')
DB_SSLMODE = os.environ.get('DB_SSLMODE')
DATA_TABLE_NAME = os.environ.get('DATA_TABLE_NAME')
APIKEY_TABLE_NAME = os.environ.get('APIKEY_TABLE_NAME')

# Connect to the TimescaleDB / PostgreSQL database
conn = psycopg2.connect(
    dbname=DB_NAME,
    user=DB_USER,
    password=DB_PASSWORD,
    host=DB_HOST,
    port=DB_PORT,
    sslmode=DB_SSLMODE
)

# Create a cursor object
cur = conn.cursor()

# Function to drop a table if it exists
def drop_table(table_name):
    try:
        cur.execute(f"DROP TABLE IF EXISTS {table_name} CASCADE;")
        print(f"Table {table_name} dropped successfully.")
    except psycopg2.Error as e:
        print(f"An error occurred while dropping table {table_name}: {e}")
    finally:
        conn.commit()

# Drop the DOKU data table and APIKEY table
drop_table(DATA_TABLE_NAME)
drop_table(APIKEY_TABLE_NAME)

# Close the cursor and the connection
cur.close()
conn.close()