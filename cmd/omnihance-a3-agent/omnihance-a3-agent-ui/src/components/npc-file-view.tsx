import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Sword,
  Shield,
  Zap,
  Gauge,
  Award,
  Clock,
  User,
  Target,
  Heart,
  Sparkles,
} from 'lucide-react';
import type { NPCFileAPIData } from '@/lib/api';

interface NPCFileViewProps {
  data: NPCFileAPIData;
}

export function NPCFileView({ data }: NPCFileViewProps) {
  const activeAttacks = data.attacks.filter(
    (attack) => attack.damage > 0 || attack.range > 0,
  );

  return (
    <div className="space-y-6">
      {/* Basic Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Name</CardTitle>
            <User className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.name}</div>
            <p className="text-xs text-muted-foreground mt-1">ID: {data.id}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Level</CardTitle>
            <Award className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.level}</div>
            <p className="text-xs text-muted-foreground mt-1">NPC Level</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">HP</CardTitle>
            <Heart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.hp}</div>
            <p className="text-xs text-muted-foreground mt-1">Health Points</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Respawn Rate</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.respawn_rate}</div>
            <p className="text-xs text-muted-foreground mt-1">seconds</p>
          </CardContent>
        </Card>
      </div>

      {/* Defense Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Defense</CardTitle>
            <Shield className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">{data.defense}</div>
            <p className="text-xs text-muted-foreground mt-1">Base Defense</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Additional Defense
            </CardTitle>
            <Shield className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">
              {data.additional_defense}
            </div>
            <p className="text-xs text-muted-foreground mt-1">Extra Defense</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Appearance</CardTitle>
            <Sparkles className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">{data.appearance}</div>
            <p className="text-xs text-muted-foreground mt-1">Visual ID</p>
          </CardContent>
        </Card>
      </div>

      {/* Speed Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Attack Speed</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">
              {data.attack_speed_low === data.attack_speed_high
                ? `${data.attack_speed_low}ms`
                : `${data.attack_speed_low}-${data.attack_speed_high}ms`}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {data.attack_speed_low === data.attack_speed_high
                ? 'Fixed Speed'
                : 'Speed Range'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Movement Speed
            </CardTitle>
            <Gauge className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">{data.movement_speed}</div>
            <p className="text-xs text-muted-foreground mt-1">
              units per second
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Attack Info</CardTitle>
            <Target className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="space-y-1">
              <div className="text-sm">
                <span className="text-muted-foreground">Type: </span>
                <span className="font-semibold">{data.attack_type_info}</span>
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">Target: </span>
                <span className="font-semibold">
                  {data.target_selection_info}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Experience */}
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Player Experience
            </CardTitle>
            <Award className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">{data.player_exp}</div>
            <p className="text-xs text-muted-foreground mt-1">
              EXP for players
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">
              Mercenary Experience
            </CardTitle>
            <Award className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-xl font-semibold">{data.mercenary_exp}</div>
            <p className="text-xs text-muted-foreground mt-1">
              EXP for mercenaries
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Defense Types */}
      {(data.blue_attack_defense > 0 ||
        data.red_attack_defense > 0 ||
        data.grey_attack_defense > 0) && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Elemental Defense
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-3">
              <div>
                <div className="text-sm text-muted-foreground">Blue Attack</div>
                <div className="text-xl font-semibold">
                  {data.blue_attack_defense}
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Red Attack</div>
                <div className="text-xl font-semibold">
                  {data.red_attack_defense}
                </div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Grey Attack</div>
                <div className="text-xl font-semibold">
                  {data.grey_attack_defense}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Attacks */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sword className="h-5 w-5" />
            Attacks ({activeAttacks.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {activeAttacks.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Attack #</TableHead>
                  <TableHead className="text-right">Range</TableHead>
                  <TableHead className="text-right">Area</TableHead>
                  <TableHead className="text-right">Damage</TableHead>
                  <TableHead className="text-right">
                    Additional Damage
                  </TableHead>
                  <TableHead className="text-right">Total Damage</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {activeAttacks.map(
                  (
                    attack: {
                      damage: number;
                      additional_damage: number;
                      range: number;
                      area: number;
                    },
                    index: number,
                  ) => {
                    const totalDamage =
                      attack.damage + attack.additional_damage;
                    return (
                      <TableRow key={index}>
                        <TableCell className="font-medium">
                          Attack {index + 1}
                        </TableCell>
                        <TableCell className="text-right">
                          {attack.range}
                        </TableCell>
                        <TableCell className="text-right">
                          {attack.area}
                        </TableCell>
                        <TableCell className="text-right">
                          {attack.damage}
                        </TableCell>
                        <TableCell className="text-right">
                          {attack.additional_damage}
                        </TableCell>
                        <TableCell className="text-right font-semibold">
                          {totalDamage}
                        </TableCell>
                      </TableRow>
                    );
                  },
                )}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              No active attacks configured
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
