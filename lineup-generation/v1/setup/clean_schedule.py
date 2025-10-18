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

weeks = data["leagueSchedule"]["weeks"]
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

# Adjust for error in JSON file
temp.pop(8)
temp = temp[:20]

for week in temp:
	schedule[week[0]] = {"startDate": week[1], "endDate": week[2], "gameSpan": week[3]}

# for week_number, info in schedule.items():
# 	print(week_number, info)

season_start = datetime.strptime("10/21/2025", "%m/%d/%Y")
game_dates = data["leagueSchedule"]["gameDates"]
cur_week = 1
game_date_format = "%m/%d/%Y %H:%M:%S"

games_in_week = defaultdict(dict)
for day in game_dates:
	if cur_week >= 20:
		break
	game_date = convert_time(day["gameDate"], game_date_format)
	week_start_date = schedule[cur_week]["startDate"]
	week_end_date = schedule[cur_week]["endDate"]
	if game_date < season_start:
		continue
	if game_date > week_end_date:
		week_start_date = schedule[cur_week + 1]["startDate"]
		schedule[cur_week]["games"] = games_in_week
		cur_week += 1
		games_in_week = defaultdict(dict)
	days_since = (game_date - week_start_date).days
	days_since = 0 if days_since == 7 or days_since == 14 else days_since
	for game in day["games"]:
		games_in_week[game["homeTeam"]["teamTricode"]][int(days_since)] = True
		games_in_week[game["awayTeam"]["teamTricode"]][int(days_since)] = True
# Handle leftover games (ie. last week)
schedule[cur_week]["games"] = games_in_week

# Convert the datetime objects to strings
for week in schedule:
	schedule[week]["startDate"] = schedule[week]["startDate"].strftime("%m/%d/%Y")
	schedule[week]["endDate"] = schedule[week]["endDate"].strftime("%m/%d/%Y")

with open("/Users/jameskendrick/Code/Projects/cv/features/lineup-generation/v1/static/schedule2025-2026.json", 'w') as f:
	json.dump(schedule, f, indent=4)