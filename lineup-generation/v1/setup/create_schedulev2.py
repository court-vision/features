import json
from datetime import datetime
from collections import defaultdict

def load_json_file(file_path):
	with open(file_path, 'r') as file:
		data = json.load(file)
	return data

# Convert time from one format to another (returning datetime object)
def convert_time(time_str, from_format="%Y-%m-%dT%H:%M:%SZ", to_format="%m/%d/%Y") -> datetime:
	return datetime.strptime(   (datetime.strptime(time_str, from_format).strftime(to_format))  , to_format   )

json_file_path = "/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v1/static/schedule_raw2025-2026.json"
data = load_json_file(json_file_path)

schedule = {}

weeks = data["leagueSchedule"]["weeks"][:21]
temp = []
for i, week in enumerate(weeks):
	week_number = week["weekNumber"]
	start_date = convert_time(week["startDate"])
	end_date = convert_time(week["endDate"]) if week_number != 17 else convert_time(weeks[i + 1]["endDate"])
	game_span = (end_date - start_date).days + 1
	temp.append((week_number, start_date, end_date, game_span))

# Adjust for all-star break
temp.pop(17)
temp.sort(key=lambda x: x[0])
for i in range(len(temp)):
	if i > 17:
		temp[i] = (temp[i][0] - 1, temp[i][1], temp[i][2], temp[i][3])

temp[17] = (temp[17][0] - 1, temp[17][1], temp[17][2], temp[17][3])
for week in temp:
	print(week)


for week in temp:
	schedule[week[0]] = {"startDate": week[1], "endDate": week[2], "gameSpan": week[3]}

# for week_number, info in schedule.items():
# 	print(week_number, info)

season_start = datetime.strptime("10/21/2025", "%m/%d/%Y")
game_dates = data["leagueSchedule"]["gameDates"]
cur_week = 1
game_date_format = "%m/%d/%Y %H:%M:%S"

new_schedule = defaultdict(list)

games_in_week = defaultdict(dict)
for day in game_dates:
	game_date = convert_time(day["gameDate"], game_date_format)
	for game in day["games"]:
		new_schedule[game_date.strftime("%m/%d/%Y")].append({"homeTeam": game["homeTeam"]["teamTricode"], "awayTeam": game["awayTeam"]["teamTricode"]})

with open("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v1/static/schedule2025-2026v2.json", 'w') as f:
	json.dump(new_schedule, f, indent=4)