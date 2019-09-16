#!/usr/bin/python3.6

# dependencies: prettytable
#    sudo pip3 install PTable

# usage:
#   python3 parser.py input_file (-o)

# input file string format:
#   common key:   <key : value>
#   private key:  [key : value]
#   end_of_log:   %%==%%

import sys
import argparse
import re
from collections import OrderedDict
import numpy as np
import copy
from prettytable import PrettyTable
import csv

def GetArgs():
    parse = argparse.ArgumentParser()
    # parse.add_argument("-i", "--input", help="input file name", type=str) # optional argument
    parse.add_argument("input", help="input file name", type=str) # compulsary argument
    parse.add_argument("-o", help="whether to output csv file or not", action="store_true") # flag argument
    args = parse.parse_args()
    global IN_FILE, OUT_FLAG
    IN_FILE = args.input
    OUT_FLAG = args.o
    ## print(IN_FILE)
    ## print(OUT_FLAG)

def ParseKeys():
    p_common_key = re.compile(r'< ?([^:=]+?) ?[:=] ?([^:=]+?) ?>')
    # p_private_key = re.compile(r'\[ ?([^:=\s]+?) ?[:=] ?([^:=\s]+?) ?\]')
    p_private_key = re.compile(r'\[ ?([^:=]+?) ?[:=] ?([^:=]+?) ?\]')
    p_end_item = re.compile(r'\%\%==\%\%')
    global in_FILE
    input = open(IN_FILE)
    line_cnt = 0
    end_cnt = 0
    dict_common_key = OrderedDict()    # stores "key-value" pair, key is the common key, value is the key's first value(as the key's default value)
    dict_private_key = OrderedDict()   # stores "key-time" pair, key is the private key, time is the times that the key appears
    list_log = []           # stores (line_cnt, discriptor, key, value) pairs, disctriptor shows whether it's common or private
    for line in input:
        ## print('line %d: %s' % (line_cnt, line), end='')
        line_cnt += 1
        if p_end_item.search(line) != None:                                   # the end item should be isolated, any log in the same line is invalid
            list_log.append((line_cnt, 'end', 'none', 'none'))
            end_cnt += 1
            ## print('catched end')
            continue
        for item in p_common_key.findall(line):
            list_log.append((line_cnt, 'common', item[0], item[1]))
            if item[0] not in dict_common_key:
                dict_common_key[item[0]] = item[1]
            ## print('common pair, %s : %s' % (item[0], item[1]))
        for item in p_private_key.findall(line):
            list_log.append((line_cnt, 'private', item[0], item[1]))
            if item[0] in dict_private_key:
                dict_private_key[item[0]] += 1
            else:
                dict_private_key[item[0]] = 1
            ## print('private pair, %s : %s' % (item[0], item[1]))
    sorted_list_private_key = sorted(dict_private_key.items(), key=lambda item:item[1], reverse=True) # sort list according to the appearance times of the private key, from the most frequent to the least
    list_log.sort(key=lambda item:item[0]) # list_log is sorted, from the first line_cnt to the last
    print(list_log)
    print(end_cnt)

    global list_field_key, list_table_rows
    list_field_key = []                             # list_field_key contains all field names of the table
    list_row_value_default = []
    list_table_rows = []                            # list_table_rows contains all the table row messages
    key_cnt = 0
    for item in dict_common_key.items():
        list_field_key.append(item[0])
        list_row_value_default.append('')
        dict_common_key[item[0]] = key_cnt          # the value of dict_common_key becomes the position of corresponding key in list_row_value
        key_cnt += 1
    for item in sorted_list_private_key:
        list_field_key.append(item[0])
        list_row_value_default.append('')
        dict_private_key[item[0]] = key_cnt         # the value of dict_private_key becomes the position of corresponding key in list_row_value
        key_cnt += 1

    list_row_value = copy.deepcopy(list_row_value_default)
    for log in list_log:
        if log[1] == 'end':
            list_table_rows.append(copy.deepcopy(list_row_value))
            ## print('new row, ', end='')
            ## print(list_row_value)
            list_row_value = copy.deepcopy(list_row_value_default)
        elif log[1] == 'common':
            list_row_value_default[dict_common_key[log[2]]] = log[3]
            list_row_value[dict_common_key[log[2]]] = log[3]
        else:
            list_row_value[dict_private_key[log[2]]] = log[3]
    print(list_table_rows)

def Output():
    global IN_FILE, OUT_FLAG, list_field_key, list_table_rows
    table = PrettyTable()
    table.field_names = list_field_key
    for row in list_table_rows:
        table.add_row(row)
    print(table)
    f = open('result.tab', 'w+')
    print(table, file = f)
    f.close()
    if OUT_FLAG:
        out_file = IN_FILE.split('.')[0] + '.csv'
        with open(out_file, 'w') as f:
            writer = csv.writer(f)
            writer.writerow(list_field_key)
            writer.writerows(list_table_rows)


if __name__ == '__main__':
    GetArgs()
    ParseKeys()
    Output()