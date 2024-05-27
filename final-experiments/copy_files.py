import os
import sys
import shutil

def copy_file_pairs(source_folder, destination_folder, num_pairs):
    # Ensure the source folder exists
    if not os.path.exists(source_folder):
        print(f"Source folder '{source_folder}' does not exist.")
        return

    # Ensure the destination folder exists, if not, create it
    if not os.path.exists(destination_folder):
        os.makedirs(destination_folder)

    # Get a list of files in the source folder
    files = os.listdir(source_folder)

    # Sort files in reversed alphabetical order
    files.sort(reverse=True)

    # Copy pairs of JSON and PEM files
    num_pairs_copied = 0
    for file_name in files:
        if file_name.endswith('.json'):
            pem_file_name = file_name[:-5] + '.pem'
            if pem_file_name in files:
                source_json = os.path.join(source_folder, file_name)
                source_pem = os.path.join(source_folder, pem_file_name)
                destination_json = os.path.join(destination_folder, file_name)
                destination_pem = os.path.join(destination_folder, pem_file_name)
                shutil.copy(source_json, destination_json)
                shutil.copy(source_pem, destination_pem)
                num_pairs_copied += 1
                if num_pairs_copied == num_pairs:
                    break

    print(f"Successfully copied {num_pairs_copied} pairs of JSON and PEM files from '{source_folder}' to '{destination_folder}'.")

if __name__ == "__main__":
    # Check if correct number of arguments are provided
    if len(sys.argv) != 4:
        print("Usage: python program_name.py source_folder destination_folder num_pairs")
        sys.exit(1)

    # Parse command-line arguments
    source_folder = sys.argv[1]
    destination_folder = sys.argv[2]
    num_pairs_to_copy = int(sys.argv[3])

    # Call the function to copy file pairs
    copy_file_pairs(source_folder, destination_folder, num_pairs_to_copy)
